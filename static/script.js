let prevCleanup = null;
let canSubmit = false;
let lastUuid = null;

async function fetchNext() {
    if (prevCleanup) {
        prevCleanup();
        prevCleanup = null;
    }

    const response = await fetch("/data.json");

    if (!response.ok) {
        const text = await response.text();
        console.error("error fetching data:", text);
        return;
    }

    const ct = response.headers.get("content-type");
    if (!ct || ct !== "application/json") {
        throw new Error(`expected json, got: ${ct}`);
    }

    try {
        const data = await response.json();
        // Store UUID from top level
        lastUuid = data.uuid;
        
        // Handle the new structure
        if (data.proto && data.proto.inputs && data.proto.inputs.length > 0) {
            renderInput(data.proto.inputs[0]);
        }
        if (data.proto && data.proto.output) {
            renderOutput(data.proto.output);
        }
    } catch (error) {
        console.error("error rendering data:", error);
    }
    
    canSubmit = true;
}

let lastInput = null;

function renderInput(input) {
    if (!input.Visualization) {
        throw new Error('Missing Visualization in input');
    }

    let elem;
    if (input.Visualization.Grid) {
        elem = renderGrid({
            rows: input.Visualization.Grid.rows,
            cols: input.Visualization.Grid.cols,
            data: input.data?.Data?.Ints?.values || []
        });
    } else {
        throw new Error(`unknown visualization type: ${JSON.stringify(input.Visualization)}`);
    }

    const div = document.querySelector(".input");
    div.replaceChildren(elem);

    // Store the input data to be submitted back later
    lastInput = input.data?.Data?.Ints?.values || [];
}

function renderOutput(output) {
    const dest = document.querySelector(".output");

    if (!output.Output?.OptionList) {
        throw new Error(`unknown output type: ${JSON.stringify(output)}`);
    }

    const cleanup = renderOneHot({
        type: "onehot",
        options: output.Output.OptionList.options.map(opt => ({
            label: opt.label,
            key: opt.hotkey
        }))
    }, dest);
    
    prevCleanup = cleanup;
    dest.classList.remove("disabled");
}

function renderGrid(input) {
    const gridContainer = document.createElement("div");
    gridContainer.className = "grid-container";

    const rows = input.rows || 0;
    const cols = input.cols || 0;
    const data = input.data || [];

    const table = document.createElement("table");
    table.className = "grid-table";

    for (let r = 0; r < rows; r++) {
        const row = document.createElement("tr");
        for (let c = 0; c < cols; c++) {
            const cell = document.createElement("td");
            const d = data[r * cols + c];
            setCellStyle(cell.style, d);
            cell.textContent = d;
            row.appendChild(cell);
        }
        table.appendChild(row);
    }

    gridContainer.appendChild(table);
    return gridContainer;
}

function setCellStyle(style, val) {
    if (val) {
        style.color = "#fff";
        style.backgroundColor = "#000";
    }
}

function renderOneHot(config, dest) {
    const container = document.createElement("div");
    container.className = "one-hot";

    const options = config.options || [];
    const buttons = {};

    for (let i = 0; i < options.length; i++) {
        const option = options[i];

        const button = document.createElement("button");
        button.textContent = option.label;

        button.addEventListener("click", () => submitExample(i));
        
        const div = document.createElement("div");
        div.appendChild(button);
        container.appendChild(div);

        if (option.key) {
            buttons[option.key] = button;
        }
    }

    const handler = function(event) {
        const button = buttons[event.key];
        if (button) {
            button.click();
            button.focus();
        }
    };
    
    document.addEventListener("keydown", handler);
    dest.replaceChildren(container);

    return function() {
        document.removeEventListener("keydown", handler);
    };
}

function submitExample(output) {
    if (!canSubmit) {
        return;
    }
    canSubmit = false;

    const dest = document.querySelector(".output");
    dest.classList.add("disabled");

    if (!lastInput) {
        throw new Error(`submitExample called but lastInput is null`);
    }

    if (!lastUuid) {
        throw new Error(`submitExample called but lastUuid is null`);
    }

    const data = {
        output: {
            optionList: {
                index: output
            }
        }
    };

    fetch(`/submit/${lastUuid}`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`failed to submit: ${response.status}`);
        }
        console.log("submitted:", data);
        fetchNext();
    })
    .catch(error => {
        console.error("error submitting:", error);
    });
}

document.getElementById("fetchDataButton").addEventListener("click", fetchNext);
fetchNext();