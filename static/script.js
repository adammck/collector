let prevCleanup = null;

// to debounce submissions.
let canSubmit = false;

async function fetchNext() {

    // clean up old event handlers, if needed.
    if (prevCleanup) {
        prevCleanup();
        prevCleanup = null;
    }

    const response = await fetch("/data.json");

    if (!response.ok) {
        // assume text if error
        const text = await response.text();
        console.error("error fetching data:", text);
        return;
    }

    // expect json on success
    const ct = response.headers.get("content-type");
    if (!ct || ct !== "application/json") {
        throw new Error(`expected json, got: ${ct}`);
    }

    try {
        const data = await response.json();
        renderInput(data.inputs[0]);
        renderOutput(data.output);
    } catch (error) {
        console("error rendering data:", error);
    }
    
    canSubmit = true;
}

let lastInput = null;

function renderInput(input) {
    const typ = input.ui.type;
    var elem;

    if (typ === "grid2d") {
        elem = renderGrid(input);
    } else {
        throw new Error(`unknown input type: ${typ}`);
    }

    const div = document.querySelector(".input");
    div.replaceChildren(elem);

    // store the input data to be submitted back later, with the output.
    lastInput = input.data;
}

function renderOutput(config) {
    const typ = config.type;

    const dest = document.querySelector(".output");

    if (typ === "onehot") {
        cleanup = renderOneHot(config, dest);
        prevCleanup = cleanup;
    } else {
        throw new Error(`unknown output type: ${typ}`);
    }

    dest.classList.remove("disabled");
}

function renderGrid(input) {
    const gridContainer = document.createElement("div");
    gridContainer.className = "grid-container";

    const rows = input.ui.grid2d.rows;
    const cols = input.ui.grid2d.cols;
    const data = input.data;

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

    const options = config.one_hot.options;
    const buttons = {};

    submit = function(index) {
        var onehot = [];
        for (let i = 0; i < options.length; i++) {
            onehot[i] = (index == i) ? 1 : 0;
        }
        submitExample(onehot);
    }

    for (let i = 0; i < options.length; i++) {
        const option = options[i];

        const button = document.createElement("button");
        button.textContent = option.label;

        button.addEventListener("click", function() { submit(i) });
        
        const div = document.createElement("div");
        div.appendChild(button);

        container.appendChild(div);

        if (option.key) {
            buttons[option.key] = button;
        }
    }

    handler = function(event) {
        const button = buttons[event.key];

        if (button) {
            button.click();
            button.focus();
        }
    }
    document.addEventListener("keydown", handler);

    dest.replaceChildren(container);

    return function() {
        document.removeEventListener("keydown", handler);
    };
}

function submitExample(output) {
    // ignore the submission if we are already processing one.
    if (!canSubmit) {
        return
    }
    canSubmit = false;

    const dest = document.querySelector(".output");
    dest.classList.add("disabled");

    if (!lastInput) {
        throw new Error(`submitExample called but lastInput is null`);
    }

    const data = {
        input: lastInput,
        output: output
    };

    fetch("/submit", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data),
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`failed to submit: ${response.status}`);
        }
        console.log("submitted:", data);
        fetchNext();
    })
    .catch(error => {
        console.error("error submitting: ", error);
    });
}

document.getElementById("fetchDataButton").addEventListener("click", function() {
    fetchNext();
});

fetchNext();
