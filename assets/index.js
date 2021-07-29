var c, ctx;
var canvasWidth, canvasHeight;
var iteration;
var colors;

function drawRec(x, y, w, h, color) {
    ctx.beginPath();
    ctx.rect(x, y, w, h);
    ctx.fillStyle = color;
    ctx.fill();
}

function drawAgent(x, y, color) {
    ctx.beginPath();
    ctx.arc(x, y, 5, 0, 2 * Math.PI, false);
    ctx.fillStyle = color;
    ctx.fill();
}

function drawAgents(agents) {
    for (i = 0; i < agents.Num; i++){
        x = agents.Agents[i].X * 5
        y = agents.Agents[i].Y * 5
        if (agents.Agents[i].ID == -1) {
            agents.Agents[i].ID += 1
        }
        color = colors[agents.Agents[i].ID % 7]
        drawAgent(x, y, color)
    }
}

function setup() {
    c = document.getElementById("canvas");
    ctx = c.getContext("2d");
    canvasWidth = c.width;
    canvasHeight = c.height;
    iteration = 0;
    colors = ["yellow", "red", "blue", "green", "violet", "orange", "cyan"]
    drawRec(0, 0, canvasWidth, canvasHeight, "#2b2828");

    // Start simulation and get initial positions and draw them
    let payload = {
        NumAgents: 6,
        World: "Static",
        BondedAgents: "[]",
        DSImode: "Fixed",
    };
    fetch("/simulation", {
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        },
        method: "POST",
        body: JSON.stringify(payload)
    }).then((response) => {
        response.text().then(function (data) {
            let result = JSON.parse(data)
            console.log(result)
            drawAgents(result)
        });
    }).catch((error) => {
        console.log(error)
    });
    
}

function startSim() {
    // TODO
    // change button to Pause and change onclick event

    // Start the animation frame after pressing start button
    window.requestAnimationFrame(moveDot)
}

// This function moves the dot down and to the right in each frame.
async function moveDot() {

    while (true) {
        // get agents
        let agents = await fetch_agents()

        if (agents.Finished) {
            console.log("Simulation ended")
            return
        }
        
        // Clear the canvas so we can draw on it again.
        drawRec(0, 0, canvasWidth, canvasHeight, "#2b2828");

        // Draw the agents.
        drawAgents(agents)

        iteration += 1;
        await sleep()
    }
}

function sleep() { 
    return new Promise(requestAnimationFrame); 
}

async function fetch_agents() {
    try {
        let response = await fetch("/simulation", {
            headers: {
                'Accept': 'application/json'
            },
            method: "GET"
            });
        return await response.json();
    } catch(e) {
        console.log(e)
    }
}