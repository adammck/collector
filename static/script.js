class StateNotifier {
    constructor(element) {
        this.element = element;
        this.currentState = 'idle';
    }

    setState(state, error = null) {
        this.currentState = state;
        
        const stateMessages = {
            idle: { class: 'info', text: 'Idle.' },
            awaiting_data: { class: 'info', text: 'Waiting for data...' },
            waiting_user: { class: 'info', text: 'Waiting for user...' },
            submitting: { class: 'info', text: 'Submitting response...' },
            server_error: { class: 'error', text: error || 'Server error' },
            client_error: { class: 'error', text: error || 'Application error' }
        };

        const stateInfo = stateMessages[state] || { class: 'info', text: 'Unknown state' };
        
        this.element.className = `state-message ${stateInfo.class}`;
        this.element.textContent = stateInfo.text;
        this.element.style.display = 'block';
    }

    clear() {
        this.element.style.display = 'none';
        this.element.textContent = '';
    }
}

class QueueManager {
    constructor(statusElement) {
        this.statusElement = statusElement;
        this.currentUUID = null;
    }
    
    updateStatus(queue) {
        if (!queue) return;
        
        const { total, active, deferred } = queue;
        
        let statusText = `Queue: ${active} active`;
        if (deferred > 0) statusText += `, ${deferred} deferred`;
        statusText += `, ${total} total`;
        
        this.statusElement.innerHTML = `
            <div class="queue-status">
                <span>${statusText}</span>
            </div>
        `;
    }
    
    async defer() {
        if (!this.currentUUID) return null;
        
        try {
            const resp = await fetch(`/defer/${this.currentUUID}`, {
                method: 'POST'
            });
            
            if (resp.ok) {
                return await resp.json();
            } else {
                throw new Error(`defer failed: ${resp.status}`);
            }
        } catch (err) {
            console.error('defer error:', err);
            throw err;
        }
    }
    
    setupKeyboardShortcuts(fetcher) {
        document.addEventListener('keydown', (e) => {
            if (e.key === 'd' && e.ctrlKey) {
                e.preventDefault();
                this.handleDefer(fetcher);
            }
            if (e.key === 'n' && e.ctrlKey) {
                e.preventDefault();
                fetcher.next();
            }
        });
    }
    
    async handleDefer(fetcher) {
        try {
            fetcher.stateNotifier.setState('awaiting_data', 'deferring item...');
            const data = await this.defer();
            
            // defer endpoint returns next item directly
            if (data && data.uuid) {
                fetcher.processNewData(data);
            } else {
                await fetcher.next();
            }
        } catch (err) {
            fetcher.handleError(err);
        }
    }
}

// Modify the Fetcher class to include state management
class Fetcher {
    constructor(inFactory, outFactory) {
        this.api = {
            fetch: "/data.json",
            submit: (uuid) => `/submit/${uuid}`
        };
        this.canSubmit = false;
        this.uuid = null;
        this.inFactory = inFactory;
        this.outFactory = outFactory;
        this.in = null;
        this.out = null;
        this.stateNotifier = new StateNotifier(document.querySelector('.state-message'));
        this.queueManager = new QueueManager(document.querySelector('.queue-status'));
        this.queueManager.setupKeyboardShortcuts(this);
    }

    async next() {
        try {
            this.stateNotifier.setState('awaiting_data');
            const resp = await this.fetchWithRetry(this.api.fetch);
            const data = await resp.json();
            
            this.processNewData(data);

        } catch (err) {
            this.handleError(err);
        }
    }

    processNewData(data) {
        if (!data.uuid || typeof data.uuid !== 'string') {
            throw new Error('invalid uuid in response');
        }
        if (!data.proto || typeof data.proto !== 'object') {
            throw new Error('missing proto in response');
        }
        
        this.uuid = data.uuid;
        this.queueManager.currentUUID = data.uuid;
        
        // update queue status
        if (data.queue) {
            this.queueManager.updateStatus(data.queue);
        }
        
        if (this.in) this.in.cleanup();
        if (this.out) this.out.cleanup();
        
        const input = data.proto.inputs?.[0];
        if (input) {
            this.validateInput(input);
            this.in = this.inFactory.create(input);
            this.in.handle(input);
        }
        
        const output = data.proto.output;
        if (output) {
            this.validateOutput(output);
            this.out = this.outFactory.create(output, i => this.submit(i));
            this.out.handle(output);
        }
        
        this.canSubmit = true;
        this.stateNotifier.setState('waiting_user');
    }

    async fetchWithRetry(url, maxAttempts = 3, options = {}) {
        let lastError;
        
        for (let attempt = 0; attempt < maxAttempts; attempt++) {
            if (attempt > 0) {
                // exponential backoff
                await new Promise(r => setTimeout(r, Math.pow(2, attempt) * 1000));
            }
            
            try {
                const resp = await fetch(url, options);
                
                if (resp.status === 408) {  // request timeout
                    lastError = new Error('timeout');
                    continue;  // retry
                }
                
                if (resp.status === 503) {  // service unavailable
                    lastError = new Error('service unavailable');
                    continue;  // retry
                }
                
                if (!resp.ok) {
                    let errorMsg;
                    try {
                        const error = await resp.json();
                        errorMsg = error.message || `http ${resp.status}`;
                    } catch {
                        errorMsg = `http ${resp.status}`;
                    }
                    throw new Error(errorMsg);
                }
                
                if (resp.headers.get("content-type") !== "application/json") {
                    throw new Error(`invalid content-type: ${resp.headers.get("content-type")}`);
                }
                
                return resp;
                
            } catch (err) {
                if (err.name === 'TypeError') {  // network error
                    lastError = err;
                    continue;  // retry
                }
                throw err;  // non-retryable
            }
        }
        
        throw lastError;
    }
    
    handleError(err) {
        console.error('fetch error:', err);
        
        // categorize error
        if (err.message.includes('timeout')) {
            this.stateNotifier.setState('server_error', 
                'no training data available - waiting...');
        } else if (err.message.includes('network') || err.name === 'TypeError') {
            this.stateNotifier.setState('client_error', 
                'network error - check connection');
        } else {
            this.stateNotifier.setState('client_error', err.message);
        }
        
        // retry after delay
        setTimeout(() => this.next(), 5000);
    }

    async submit(idx) {
        if (!this.canSubmit) return;
        if (!this.in?.lastInput) throw new Error('no input data');
        if (!this.uuid) throw new Error('no uuid');

        this.canSubmit = false;
        this.out?.disable();

        try {
            this.in?.clear();
            this.out?.clear();
            
            this.stateNotifier.setState('submitting');
            const resp = await this.fetchWithRetry(this.api.submit(this.uuid), 3, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    output: { optionList: { index: idx } }
                })
            });
            
            await this.next();

        } catch (err) {
            this.handleError(err);
            throw err;
        }
    }

    validateInput(input) {
        if (!input.Visualization || typeof input.Visualization !== 'object') {
            throw new Error('invalid input visualization');
        }
        if (input.Visualization.Grid) {
            const { rows, cols } = input.Visualization.Grid;
            if (!Number.isInteger(rows) || rows <= 0 || !Number.isInteger(cols) || cols <= 0) {
                throw new Error('invalid grid dimensions');
            }
            if (!input.data?.Data?.Ints?.values || !Array.isArray(input.data.Data.Ints.values)) {
                throw new Error('invalid grid data');
            }
            if (input.data.Data.Ints.values.length !== rows * cols) {
                throw new Error('grid data length mismatch');
            }
        }
    }

    validateOutput(output) {
        if (!output.Output?.OptionList || !Array.isArray(output.Output.OptionList.options)) {
            throw new Error('invalid option list');
        }
        for (const opt of output.Output.OptionList.options) {
            if (!opt.label || typeof opt.label !== 'string') {
                throw new Error('invalid option label');
            }
        }
    }
}

class Input {
    constructor(el) {
        this.el = el;
        this.lastInput = null;
    }
    
    handle(input) {
        throw new Error('not implemented');
    }

    clear() {
        this.el.replaceChildren();
        this.lastInput = null;
    }

    cleanup() {}
}

class GridInput extends Input {
    handle(input) {
        const { rows, cols } = input.Visualization.Grid;
        const vals = input.data.Data.Ints.values;
        
        const table = document.createElement("table");
        table.className = "grid-table";

        for (let r = 0; r < rows; r++) {
            const row = document.createElement("tr");
            for (let c = 0; c < cols; c++) {
                const cell = document.createElement("td");
                const val = vals[r * cols + c];
                if (val) {
                    cell.style.color = "#fff";
                    cell.style.backgroundColor = "#000";
                }
                cell.textContent = val;
                row.appendChild(cell);
            }
            table.appendChild(row);
        }

        const grid = document.createElement("div");
        grid.className = "grid-container";
        grid.appendChild(table);
        
        this.el.replaceChildren(grid);
        this.lastInput = vals;
    }
}

class Output {
    constructor(el) {
        this.el = el;
        this.cleanups = null;
    }
    
    handle(output) {
        throw new Error('not implemented');
    }
    
    disable() {
        this.el.classList.add("disabled");
    }
    
    enable() {
        this.el.classList.remove("disabled");
    }
    
    cleanup() {
        if (this.cleanups) {
            this.cleanups();
            this.cleanups = null;
        }
    }

    clear() {
        this.cleanup();
        this.el.replaceChildren();
    }
}


class OptionList extends Output {
    constructor(el, onSubmit) {
        super(el);
        this.onSubmit = onSubmit;
    }

    handle(output) {
        this.cleanup();

        const container = document.createElement("div");
        container.className = "one-hot";
        const keys = new Map();

        output.Output.OptionList.options.forEach((opt, i) => {
            const btn = document.createElement("button");
            btn.textContent = opt.label;
            btn.addEventListener("click", () => this.onSubmit(i));

            const div = document.createElement("div");
            div.appendChild(btn);
            container.appendChild(div);

            if (opt.hotkey) keys.set(opt.hotkey, btn);
        });

        const handler = (e) => {
            const btn = keys.get(e.key);
            if (btn) {
                btn.click();
                btn.focus();
            }
        };

        document.addEventListener("keydown", handler);
        this.el.replaceChildren(container);
        this.enable();

        this.cleanups = () => document.removeEventListener("keydown", handler);
    }
}

class InFactory {
    constructor(el) {
        this.el = el;
    }
    
    create(input) {
        if (input.Visualization?.Grid) {
            return new GridInput(this.el);
        }
        throw new Error(`unknown input type`);
    }
}

class OutFactory {
    constructor(el) {
        this.el = el;
    }
    
    create(output, onSubmit) {
        if (output.Output?.OptionList) {
            return new OptionList(this.el, onSubmit);
        }
        throw new Error(`unknown output type`);
    }
}

const inFactory = new InFactory(document.querySelector(".input"));
const outFactory = new OutFactory(document.querySelector(".output"));
const fetcher = new Fetcher(inFactory, outFactory);

document.getElementById("fetchDataButton").addEventListener("click", () => fetcher.next());
document.getElementById("deferButton").addEventListener("click", () => fetcher.queueManager.handleDefer(fetcher));
fetcher.next();
