const playerName = document.getElementById("clientName").innerText; // 抓取玩家名字
let isPlaying = false
let numOfClients
// Add an event listener to the color input
const colorInput = document.getElementById('colourInput');
colorInput.addEventListener('input', updateCurrentColor);
colorInput.addEventListener('click', ()=>{colorOptions.forEach(option => option.classList.remove('selected'));});

// Add event listeners to color options
const colorOptions = document.querySelectorAll('.color-option');

// Function to update the current color display
function updateCurrentColor(event) {
    const currentColorDisplay = document.getElementById('currentColorDisplay');
    const selectedColor = event.target.value || event.target.style.backgroundColor;
    
    currentColorDisplay.style.backgroundColor = selectedColor;
}

colorOptions.forEach(colorOption => {
    colorOption.addEventListener('click', function (event) {
        // Remove 'selected' class from all color options
        colorOptions.forEach(option => option.classList.remove('selected'));

        // Add 'selected' class to the clicked color option
        colorOption.classList.add('selected');

        // Update the border for the current color option
        updateCurrentColor(event)
    });
});
let color = "#000000";

// strokeWidth 
let strokeWidth = 5;
const strokeWidthInput = document.getElementById("strokeWidthInput")
strokeWidthInput.addEventListener("change", ()=>{strokeWidth = parseInt(strokeWidthInput.value)})

// penStyle
let penStyle = 1;
const allPenStyle = document.querySelectorAll(".typeOfPen")
// setting penStyle visulization
// and to handle pen type selection
const selectPenType = (event) => {
    // Remove the 'selected' class from all pen types
    allPenStyle.forEach(pen => pen.classList.remove('selected'));

    // Add the 'selected' class to the clicked pen type
    const selectedPen = event.currentTarget;
    selectedPen.classList.add('selected');

    // Get the pen name attribute and use it as needed
    penStyle = parseInt(selectedPen.getAttribute('name'));
    console.log('Selected Pen Name:', penStyle);
};

// Add click event listener to each pen type
allPenStyle.forEach(pen => {
    pen.addEventListener('click', selectPenType);
});

// progress bar
const progressBar = document.getElementById("progressFill");

let latestPoint;
let drawing = false;
let scoreBoard = {} // to memorize the scores for players
let cur_next_painter_and_questions = ["", "", "", ""] // just a list to store current and the next painter
let roomMaster = ""

const canvas = document.getElementById("canvas");
const context = canvas.getContext("2d");
const startBtn = document.getElementById("startBtn");
startBtn.addEventListener("click", GS);

function GS() {
    console.log("GS")
    isGameOver = false
    // reset score board
    Object.keys(scoreBoard).forEach(k => {
        scoreBoard[k] = 0
    });
    playerBlocks.forEach( playerBlock => {
        playerBlock.querySelector(".score").innerText = 0
    })
    var jsonObject = {"Type":"GS"};
    var jsonString = JSON.stringify(jsonObject);
    socket.send(jsonString)
}

const RO = () => {
    var jsonObject = {"Type":"RO"};
    var jsonString = JSON.stringify(jsonObject);
    if (playerName == cur_next_painter_and_questions[0]) {
        socket.send(jsonString)   
    }
}

const RSK = () => {
    var jsonObject = {"Type":"RSK"};
    var jsonString = JSON.stringify(jsonObject);
    if (playerName == cur_next_painter_and_questions[0]) {
        socket.send(jsonString)   
    }
}

const defaultStroke = (origin, destination, color, strokeWidth) => {
    context.moveTo(origin[0], origin[1]);
    context.strokeStyle = color;
    context.lineWidth = strokeWidth;
    context.lineCap = "round";
    context.lineJoin = "round";
    context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
    context.lineTo(destination[0], destination[1]);
    context.stroke();
};

const defaultEraser = (origin, destination, strokeWidth) => {
    context.moveTo(origin[0], origin[1]);
    context.lineWidth = strokeWidth + 2;
    context.lineCap = "round";
    context.lineJoin = "round";
    context.globalCompositeOperation = "destination-out"; // Use 'destination-out' for erasing
    context.lineTo(destination[0], destination[1]);
    context.stroke();
};

async function dotStroke() {
    console.log("dotStroke");
    context.beginPath();

    switch (penStyle) {
        case 1: // Normal line
            context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
            context.fillStyle = color;
            context.arc(latestPoint[0], latestPoint[1], strokeWidth / 2, 0, 2 * Math.PI);
            context.fill();
            break;

        case 2: // Bristle brush
            const bristleCount = Math.round(strokeWidth / 3);
            const gap = strokeWidth / bristleCount;
            for (let i = 0; i < bristleCount; i++) {
                context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
                context.fillStyle = color;
                context.beginPath();
                context.arc(
                    latestPoint[0] + i * gap,
                    latestPoint[1],
                    2, // Fixed size for bristle brush
                    0,
                    2 * Math.PI
                );
                context.fill();
            }
            break;

        case 3: // Crayon effect
            context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
            context.lineJoin = "round"; // Set line join to round for smoother connections between lines
            context.lineCap = "round"; // Set line cap to round for a rounded end

            for (let i = 0; i < 3; i++) {
                const offsetX = getRandomInt(-strokeWidth / 2, strokeWidth / 2);
                const offsetY = getRandomInt(-strokeWidth / 2, strokeWidth / 2);
                const crayonX = latestPoint[0] + offsetX;
                const crayonY = latestPoint[1] + offsetY;

                context.fillStyle = color;
                context.beginPath();
                context.arc(crayonX, crayonY, strokeWidth / 2, 0, 2 * Math.PI);
                context.fill();
            }
            break;

        case 4: // Spray effect
            context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
            context.lineWidth = strokeWidth;
            context.fillStyle = color;
            context.lineJoin = 'round';
            context.lineCap = 'round';

            const density = 50;
            const radius = strokeWidth;

            for (let i = density; i--; ) {
                const offsetX = getRandomInt(-radius, radius);
                const offsetY = getRandomInt(-radius, radius);
                const sprayX = latestPoint[0] + offsetX;
                const sprayY = latestPoint[1] + offsetY;

                context.fillRect(sprayX, sprayY, 1, 1);
            }
            break;

        case 5: // Eraser
            context.globalCompositeOperation = "destination-out"; // Erasing
            context.fillStyle = color;
            context.arc(latestPoint[0], latestPoint[1], strokeWidth / 2, 0, 2 * Math.PI);
            context.fill();
            break;
    }

    console.log("latestPoint: ",latestPoint, "color: ", color, "strokeWidth: ", strokeWidth, "penStyle: ", penStyle);
};

function send(type, content, newPoint, color, strokeWidth, penStyle) {
    console.log(type, content, newPoint, color, strokeWidth, penStyle)
    switch (type) {
        case "sys":
            var jsonObject = {"Type":type, "Payload":{"Content": content}};
            var jsonString = JSON.stringify(jsonObject);
            socket.send(jsonString)
            break;

        case "chat":
            var jsonObject = {"Type":type, "Payload":{"Content":content}};
            var jsonString = JSON.stringify(jsonObject);
            socket.send(jsonString)
            break;

        case "mouseDown": 
            var jsonObject = {"Type":type, "Payload":{"Content":content, "NewPoint":newPoint, "StrokeWidth":strokeWidth,
                              "Color":color, "PenStyle":penStyle}};
            var jsonString = JSON.stringify(jsonObject);
            socket.send(jsonString)
            break;

        case "draw":
            var jsonObject = {"Type":type, "Payload":{"NewPoint":newPoint}};
            var jsonString = JSON.stringify(jsonObject);
            socket.send(jsonString)
            break;
        

    }
}

const continueStroke = (newPoint) => {
    console.log("continueStroke");
    draw(newPoint);
    // Send drawing data to the server
    send("draw", "", newPoint)
};

function draw(newPoint) {
    context.beginPath();

    switch (penStyle) {
    case 1: // Normal line
        defaultStroke(latestPoint, newPoint, color, strokeWidth);
        break;

    case 2: // Bristle brush
        const bristleCount = Math.round(strokeWidth / 3);
        const gap = strokeWidth / bristleCount;
        for (let i = 0; i < bristleCount; i++) {
            defaultStroke(
                [latestPoint[0] + i * gap, latestPoint[1]],
                [newPoint[0] + i * gap, newPoint[1]],
                color,
                2
            );
        }
        break;

    case 3: // Crayon effect
        context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
        context.lineJoin = "round"; // Set line join to round for smoother connections between lines
        context.lineCap = "round"; // Set line cap to round for a rounded end

        for (let i = 0; i < 3; i++) {
            const offsetX = getRandomInt(-strokeWidth / 2, strokeWidth / 2);
            const offsetY = getRandomInt(-strokeWidth / 2, strokeWidth / 2);
            const crayonX = newPoint[0] + offsetX;
            const crayonY = newPoint[1] + offsetY;

            context.fillStyle = color;
            context.beginPath();
            context.arc(crayonX, crayonY, strokeWidth / 2, 0, 2 * Math.PI);
            context.fill();
        }
        break;

    case 4: // Spray effct
        context.globalCompositeOperation = "source-over"; // Use 'source-over' for drawing
        context.lineWidth = strokeWidth;
        context.fillStyle = color;
        context.lineJoin = 'round';
        context.lineCap = 'round';

        const density = 50;
        const radius = strokeWidth;

        for (let i = density; i--; ) {
            const offsetX = getRandomInt(-radius, radius);
            const offsetY = getRandomInt(-radius, radius);
            const sprayX = newPoint[0] + offsetX;
            const sprayY = newPoint[1] + offsetY;

            context.fillRect(sprayX, sprayY, 1, 1);
        }
        break;
    
    case 5: // Eraser
        defaultEraser(latestPoint, newPoint, strokeWidth);
        break;
    }

    latestPoint = newPoint;
    console.log("newPoint:", newPoint, "color: ", color, "strokeWidth: ", strokeWidth );
}

function getRandomInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

// drawing logic
const ctx = canvas.getContext("2d");

const mouse = {
    x: 0, y: 0,
};

function normalizeMouseCoords(event, bounds) {
    mouse.x = event.pageX - bounds.left - scrollX;
    mouse.y = event.pageY - bounds.top - scrollY;

    mouse.x /= bounds.width;
    mouse.y /= bounds.height;

    mouse.x *= canvas.width;
    mouse.y *= canvas.height;

    mouse.x = Math.round(mouse.x);
    mouse.y = Math.round(mouse.y);
}

function mouseEvent(event) {
    if (playerName != cur_next_painter_and_questions[0]) {
        return;
    }
    var bounds = canvas.getBoundingClientRect();
    normalizeMouseCoords(event, bounds);

    if (event.type === "mousedown" && event.target === canvas) {
        drawing = true;
        progressFill.style.opacity = "30%";
        console.log("mouseDown")
        latestPoint = [mouse.x, mouse.y]
        console.log(latestPoint);
        color = currentColorDisplay.style.backgroundColor;
        send("mouseDown", "", latestPoint, color, strokeWidth, penStyle);
        event.preventDefault();
        dotStroke();
    } else if (event.type === "mousemove" && event.target === canvas) {
        if (!drawing) {
            return;
        }
        console.log("mousemove");
        continueStroke([mouse.x, mouse.y]);
    }
}

const endStroke = () => {console.log("endStroke"); drawing = false; progressFill.style.opacity = "50%";}
canvas.addEventListener("mousemove", mouseEvent);
canvas.addEventListener("mousedown", mouseEvent);
canvas.addEventListener("mouseup", endStroke);
canvas.addEventListener("mouseout", endStroke);

// Receive data from the server
const sysChat = document.getElementById("sysChat")
const publicChat = document.getElementById("publicChat")
const sysChatDiv = sysChat.querySelector("#sysChatContent")
const publicChatDiv = publicChat.querySelector("#publicChatContent")
const overlay = sysChat.querySelector("#overlay")
let sysPrevMsg = document.getElementById("sysPlaceholder")
let publicPrevMsg = document.getElementById("publicPlaceholder")

// varibles to update the score board
let guessed = {}

// get memberBar
const memberBar = document.getElementById("memberBar");

// Create an array of player blocks
let playerBlocks = Array.from(memberBar.getElementsByClassName("playerBlock"));

// variables that control the display of transitions
let isGameOver = false
const waitMember = document.getElementById("wait-member")
const gameOverWaitMember = document.getElementById("game-over-wait-member")
const waitStart = document.getElementById("wait-start")
const gameOverWaitStart = document.getElementById("game-over-wait-start")
const roundStart = document.getElementById("round-start")
const chooseQuestion = document.getElementById("choose-question")
const questionOne = chooseQuestion.querySelector("#question-one")
const questionTwo = chooseQuestion.querySelector("#question-two")
questionOne.querySelector("button").addEventListener("click", gameStart)
questionTwo.querySelector("button").addEventListener("click", gameStart)
const roundOver = document.getElementById("round-over")
const roundSkip = document.getElementById("round-skip")
const gameOver = document.getElementById("game-over")
const restartBtn = document.getElementById("restartBtn")
restartBtn.addEventListener("click", GS)
const firstPlace = document.getElementById("podium").querySelector(".first")
const secondPlace = document.getElementById("podium").querySelector(".second")
const thirdPlace = document.getElementById("podium").querySelector(".third")

function gameStart(event) {
    console.log("CS")
    let question = event.target.previousElementSibling.innerText
    let index = event.target.nextSibling.innerText
    console.log("Question:", question)
    var jsonObject = {"Type":"CS", "Payload":{"Content":question + "@" + index}};
    var jsonString = JSON.stringify(jsonObject);
    socket.send(jsonString)   
    chooseQuestion.style.display = "none"
}

socket.onmessage = (event) => {
    console.log(event.data)
    const jsonData = JSON.parse(event.data)
    console.log(jsonData)
    switch (jsonData.type) {
        case "sys":
            // Display the string based on the prefix
            // C => Common, G => Guessed
            const sysMessageElement = document.createElement('div');
            // so that timeStamp can be attached to it as absolute element
            sysMessageElement.style.position = "relative"
            let sysNameAndContent
            sysNameAndContent = jsonData.payload.content
            if (sysNameAndContent.charAt(0) === "G") {
                // name```timeStamp
                const name = sysNameAndContent.slice(1).split("```")[0]
                const timeStamp = sysNameAndContent.slice(1).split("```")[1]

                const guessSpan = document.createElement("span");
                guessSpan.classList.add("guess-content")

                // add timeStamp
                const timeStampSpan = document.createElement("span");
                timeStampSpan.innerText = timeStamp
                timeStampSpan.classList.add("guess-timeStampSpan")

                sysMessageElement.appendChild(guessSpan)
                sysMessageElement.appendChild(timeStampSpan)
                if (name === playerName) {
                    guessSpan.innerText = "你猜對了!"
                    // 猜對後就要lock input
                    sysChatInput.disabled = true
                    overlay.style.display = "block"
                } else {
                    guessSpan.innerText = name + " 猜對了!"
                }

                // add player to guessed object
                guessed[name] = NaN
                playerBlocks.forEach( playerBlock => {
                    if ( playerBlock.querySelector(".name").innerText === sysNameAndContent.slice(1) ) {
                        playerBlock.querySelector(".guessed").style.display = "block"
                    }
                })

            } else if (sysNameAndContent.charAt(0) === "C") {
                // name```content```timeStamp
                const name = sysNameAndContent.slice(1).split("```")[0]
                const content = sysNameAndContent.slice(1).split("```")[1]
                const timeStamp = sysNameAndContent.slice(1).split("```")[2]
                // name
                const nameSpan = document.createElement("span");
                nameSpan.innerText = name
                nameSpan.classList.add("name-span")
                // content
                const contentSpan = document.createElement("span");
                contentSpan.innerText = " " + content
                contentSpan.classList.add("content-span")
                // timeStamp
                const timeStampSpan = document.createElement("span");
                timeStampSpan.innerText = timeStamp
                timeStampSpan.classList.add("timeStampSpan")

                sysMessageElement.appendChild(nameSpan)
                sysMessageElement.appendChild(contentSpan)
                sysMessageElement.appendChild(timeStampSpan)
            } 

            // there is not insertAfter, because it can be done as follows
            sysChatDiv.insertBefore(sysMessageElement, sysPrevMsg.nextSibling);
            sysPrevMsg = sysMessageElement
            sysChatDiv.scrollTop = sysChatDiv.scrollHeight; // scroll down to the latest message
            break;

        case "chat":
            const publicMessageElement = document.createElement("div");
            // so that timeStamp can be attached to it as absolute element
            publicMessageElement.style.position = "relative"
            let chatNameAndContent
            chatNameAndContent = jsonData.payload.content
            // Split the string based on the "```" delimiters
            // J => Join, L => Leave, 沒標示 => 一般訊息
            if (chatNameAndContent.charAt(0) === "J") {
                // name```timeStamp
                playerBlocks = Array.from(memberBar.getElementsByClassName("playerBlock"));
                const name = jsonData.payload.content.slice(1).split("```")[0];
                const timeStamp = jsonData.payload.content.slice(1).split("```")[1];
                // name and content
                const joinSpan = document.createElement("span");
                joinSpan.classList.add("join-content")
                joinSpan.innerText = name + " JOIN!"
                // timeStamp
                const timeStampSpan = document.createElement("span");
                timeStampSpan.innerText = timeStamp
                timeStampSpan.classList.add("join-timeStampSpan")

                publicMessageElement.appendChild(joinSpan)
                publicMessageElement.appendChild(timeStampSpan)
            } else if (chatNameAndContent.charAt(0) === "L") {
                // name```timeStamp
                playerBlocks = Array.from(memberBar.getElementsByClassName("playerBlock"));
                const name = jsonData.payload.content.slice(1).split("```")[0];
                const timeStamp = jsonData.payload.content.slice(1).split("```")[1];
                // name and content
                const leaveSpan = document.createElement("span");
                leaveSpan.classList.add("leave-content")
                leaveSpan.innerText = name + " LEAVE!"
                // timeStamp
                const timeStampSpan = document.createElement("span");
                timeStampSpan.innerText = timeStamp
                timeStampSpan.classList.add("leave-timeStampSpan")

                publicMessageElement.appendChild(leaveSpan)
                publicMessageElement.appendChild(timeStampSpan)
            } else {
                // name```content```timeStamp
                const name = chatNameAndContent.split("```")[0]
                const content = chatNameAndContent.split("```")[1]
                const timeStamp = chatNameAndContent.split("```")[2]
                // name
                const nameSpan = document.createElement("span");
                nameSpan.innerText = name
                nameSpan.classList.add("name-span")
                // content
                const contentSpan = document.createElement("span");
                contentSpan.innerText = " " + content
                contentSpan.classList.add("content-span")
                // timeStamp
                const timeStampSpan = document.createElement("span");
                timeStampSpan.innerText = timeStamp
                timeStampSpan.classList.add("timeStampSpan")

                publicMessageElement.appendChild(nameSpan)
                publicMessageElement.appendChild(contentSpan)
                publicMessageElement.appendChild(timeStampSpan)
            }
            
            publicChatDiv.insertBefore(publicMessageElement, publicPrevMsg.nextSibling);
            publicPrevMsg = publicMessageElement
            publicChatDiv.scrollTop = publicChatDiv.scrollHeight;
            break;

        case "mouseDown":
            if (playerName != cur_next_painter_and_questions[0]) {
                latestPoint = jsonData.payload.newPoint;
                color = jsonData.payload.color;
                strokeWidth = jsonData.payload.strokeWidth;
                penStyle = jsonData.payload.penStyle;
                dotStroke();
            }
            break;
            
        case "draw":
            if (playerName != cur_next_painter_and_questions[0]) {
                draw(jsonData.payload.newPoint);
            }
            break;
        
        case "score":
            for (let key in jsonData.dict) {
                if (key in scoreBoard) {
                    if (jsonData.dict[key] != -1) {
                        scoreBoard[key] = jsonData.dict[key]
                    } else {
                        delete scoreBoard[key]
                    }
                } else {
                    scoreBoard[key] = jsonData.dict[key]
                }
            }
            console.log("score board: ", scoreBoard)

            // Sort the array in reverse order based on the values
            let nameScoreArray = Object.entries(scoreBoard);
            // sort in descending order
            nameScoreArray.sort((a, b) => b[1] - a[1]);
            
            // Iterate through each player block and update the content
            let i = 0 // init the index

            playerBlocks.forEach(playerBlock => {
                // get all child elements
                const playerNameElement = playerBlock.querySelector(".name");
                const playerScoreElement = playerBlock.querySelector(".score");
                const painterDiv = playerBlock.querySelector(".painter");
                const guessedDiv = playerBlock.querySelector(".guessed");
                const roomMasterDiv = playerBlock.querySelector(".room-master");

                if (i < nameScoreArray.length) {
                    playerBlock.classList.remove("emptyPlayerBlock")
                    playerBlock.classList.remove("lastPlayerBlock")
                    
                    // reset the name and score
                    playerNameElement.innerText = nameScoreArray[i][0]
                    playerScoreElement.innerText = nameScoreArray[i][1]

                    if (playerNameElement.innerText === cur_next_painter_and_questions[0]) {
                        painterDiv.style.display = "block"
                    } else {
                        painterDiv.style.display = "none"
                    }

                    if (playerNameElement.innerText in guessed) {
                        guessedDiv.style.display = "block"
                    } else {
                        guessedDiv.style.display = "none"
                    }

                    if (playerNameElement.innerText === roomMaster) {
                        roomMasterDiv.style.display = "block"
                    } else {
                        roomMasterDiv.style.display = "none"
                    }

                    i+=1
                } else {
                    playerBlock.classList.add("emptyPlayerBlock")
                    
                    // reset the name and score
                    playerNameElement.innerText = "Empty"
                    playerScoreElement.innerText = ""

                    painterDiv.style.display = "none"
                    guessedDiv.style.display = "none"
                    roomMasterDiv.style.display = "none"
                }
            });
            
            // add class lastPlayerBlock to the last element
            playerBlocks[playerBlocks.length-1].classList.add("lastPlayerBlock")
            
            break
        
        case "roomMaster":
            console.log(jsonData.payload.content)
            roomMaster = jsonData.payload.content.split("```")[0]
            numOfClients = jsonData.payload.content.split("```")[1]
            console.log("enter roomaster, current members:", numOfClients)

            // display of toggle switch
            if ( playerName === roomMaster ) {
                toggleContainer.style.display = "flex"
            } else {
                toggleContainer.style.display = "none"
            }

            // update score board
            playerBlocks.forEach( playerBlock => {
                if ( playerBlock.querySelector(".name").innerText === roomMaster ) {
                    playerBlock.querySelector(".room-master").style.display = "block"
                }
            })

            // 如果遊戲還沒開始
            if ( !isPlaying ) {
                if ( isGameOver ) {
                    if (roomMaster === playerName) {
                        if ( numOfClients >= 2 ) {
                            gameOverWaitStart.style.display = "none"
                            restartBtn.style.display = "block"
                            gameOverWaitMember.style.display = "none"
                        } else {
                            gameOverWaitStart.style.display = "none"
                            restartBtn.style.display = "none"
                            gameOverWaitMember.style.display = "flex"
                        }
                    } else {
                        gameOverWaitStart.innerHTML =  "等待<b>" + roomMaster + "</b>開始"
                        gameOverWaitStart.style.display = "flex"
                    }
                } else {
                    if (roomMaster === playerName) {
                        waitStart.style.display = "none"
                        if ( numOfClients >= 2 ) {
                            waitMember.style.display = "none"
                            startBtn.style.display = "block"
                        } else {
                            waitMember.style.display = "flex"
                            startBtn.style.display = "none"
                        }
                    } else {
                        startBtn.style.display = "none"
                        waitStart.innerHTML = "等待<b>" + roomMaster + "</b>開始"
                        waitStart.style.display = "flex"
                    }
                }
            }

            break
        
        case "GS": // Game Start => roomMaster press the start button
            isGameOver = false
            console.log("GS")
            // 寄送GS之後回RS要設定遊戲開始
            // clear canvas
            context.clearRect(0, 0, canvas.width, canvas.height);
            isPlaying = true
            isGameOver = false
            gameOver.style.display = "none"
            restartBtn.style.display = "none"
            gameOverWaitStart.style.display = "none"
            gameOverWaitMember.style.display = "none"
            waitMember.style.display = "none"
            startBtn.style.display = "none"
            waitStart.style.display = "none"
            roundOver.style.display = "none"
            roundSkip.style.display = "none"

            cur_next_painter_and_questions = jsonData.payload.content.split("@")
            console.log(cur_next_painter_and_questions[0], cur_next_painter_and_questions[1])

            // update the painter icon
            playerBlocks.forEach( playerBlock => {
                if ( playerBlock.querySelector(".name").innerText === cur_next_painter_and_questions[0] ) {
                    playerBlock.querySelector(".painter").style.display = "block"
                } else {
                    playerBlock.querySelector(".painter").style.display = "none"
                }
            })

            // 畫家選擇 其餘玩家出現round start畫面
            if (playerName === cur_next_painter_and_questions[0]) {
                // shows up the options to choose
                questionOne.querySelector("span").innerText = cur_next_painter_and_questions[2]
                questionOne.querySelector("p").innerText = cur_next_painter_and_questions[3]
                questionTwo.querySelector("span").innerText = cur_next_painter_and_questions[4]
                questionTwo.querySelector("p").innerText = cur_next_painter_and_questions[5]
                chooseQuestion.style.display = "flex"
            } else {
                // 其餘玩家等待畫面
                roundStart.innerHTML = "等待<b>" + cur_next_painter_and_questions[0] + "</b>出題"
                roundStart.style.display = "flex"
            }

            
            //window.requestAnimationFrame(timestamp => step(timestamp, 8000, RO)); // 8000ms
            console.log("RS start counting")
            countdownTimer(8, RSK, ticker)

            break
        
        case "RS": // Round Start
            if (isGameOver) {
                console.log("Game is Over, RS blocked by front end")
                break 
            }
            console.log("RS")
            // 寄送GS之後回RS要設定遊戲開始
            // clear canvas
            context.clearRect(0, 0, canvas.width, canvas.height);
            isPlaying = true
            isGameOver = false
            gameOver.style.display = "none"
            restartBtn.style.display = "none"
            gameOverWaitStart.style.display = "none"
            gameOverWaitMember.style.display = "none"
            waitMember.style.display = "none"
            startBtn.style.display = "none"
            waitStart.style.display = "none"
            roundOver.style.display = "none"
            roundSkip.style.display = "none"

            cur_next_painter_and_questions = jsonData.payload.content.split("@")
            console.log(cur_next_painter_and_questions[0], cur_next_painter_and_questions[1])

            // update the painter icon
            playerBlocks.forEach( playerBlock => {
                if ( playerBlock.querySelector(".name").innerText === cur_next_painter_and_questions[0] ) {
                    playerBlock.querySelector(".painter").style.display = "block"
                } else {
                    playerBlock.querySelector(".painter").style.display = "none"
                }
            })

            // 畫家選擇 其餘玩家出現round start畫面
            if (playerName === cur_next_painter_and_questions[0]) {
                // shows up the options to choose
                questionOne.querySelector("span").innerText = cur_next_painter_and_questions[2]
                questionOne.querySelector("p").innerText = cur_next_painter_and_questions[3]
                questionTwo.querySelector("span").innerText = cur_next_painter_and_questions[4]
                questionTwo.querySelector("p").innerText = cur_next_painter_and_questions[5]
                chooseQuestion.style.display = "flex"
            } else {
                // 其餘玩家等待畫面
                roundStart.innerHTML = "等待<b>" + cur_next_painter_and_questions[0] + "</b>出題"
                roundStart.style.display = "flex"
            }

            
            //window.requestAnimationFrame(timestamp => step(timestamp, 8000, RO)); // 8000ms
            console.log("RS start counting")
            countdownTimer(8, RSK, ticker)

            break

        case "CS": // Confirmed Start => start drawing and guessing
            console.log("CS")
            stopCountingDown()
            chooseQuestion.style.display = "none"
            roundStart.style.display = "none"
            
            // unlock the sysChat
            if (playerName != cur_next_painter_and_questions[0]) {
                sysChatInput.disabled = false
                overlay.style.display = "none"
            }

            // Start the countdown with a customized duration
            //window.requestAnimationFrame(timestamp => step(timestamp, 90000, RO));
            console.log("CS start counting")
            countdownTimer(90, RO, ticker)
            break

        case "RO": // Round Over
            // display the transition view and clear the canvas
            console.log("RO!");
            
            reset()

            // 判斷是否大家都對(1) 或是大家都沒答對(2) 或是其他(0)
            console.log("正確率", jsonData.payload.content)
            if (jsonData.payload.content === "1") {
                roundOver.innerText = "全部答對\n中場休息"
            } else if ( jsonData.payload.content === "2" ) {
                roundOver.innerText = "無人答對\n中場休息"
            } else {
                roundOver.innerText = "中場休息"
            }

            roundOver.style.display = "flex"
            
            //window.requestAnimationFrame(timestamp => step(timestamp, 4000, ()=>{}));
            console.log("RO start counting")
            countdownTimer(4, ()=>{}, ticker)
        
            break

        case "RSK": // Round Over
            // display the transition view and clear the canvas
            if (isGameOver) {
                console.log("Game is Over, RSK blocked by front end")
                break 
            }
            console.log("RSK!");

            reset()

            chooseQuestion.style.display = "none"
            roundStart.style.display = "none"


            // 如果你是錯過選擇回合的畫家
            if (playerName === cur_next_painter_and_questions[0]) {
                roundSkip.innerText = "你錯過了回合"
                roundSkip.style.display = "flex"
            } else {
                // 你是其他玩家
                roundSkip.innerText = cur_next_painter_and_questions[0] + "錯過了回合"
                roundSkip.style.display = "flex"
            }

            //window.requestAnimationFrame(timestamp => step(timestamp, 4000, ()=>{}));
            console.log("RSK start counting")
            countdownTimer(4, ()=>{}, ticker)
            
            break

        case "GO":
            console.log("GO")
            isPlaying = false
            // reset cur_next_painter_and_questions
            cur_next_painter_and_questions = ["", "", "", ""]
            reset() // just for round reset
            // other reset when game is over
            waitStart.style.display = "none"
            roundStart.style.display = "none"
            chooseQuestion.style.display = "none"
            roundOver.style.display = "none"
            roundSkip.style.display = "none"

            // display the podium
            let podiumArray = jsonData.payload.content.split("```")
            firstPlace.querySelector(".name").innerText = podiumArray[0]
            firstPlace.querySelector(".score").innerText = podiumArray[1]
            if (podiumArray.length >= 4) {
                secondPlace.querySelector(".name").innerText = podiumArray[2]
                secondPlace.querySelector(".score").innerText = podiumArray[3]
            } 
            if (podiumArray.length == 6) {
                thirdPlace.querySelector(".name").innerText = podiumArray[4]
                thirdPlace.querySelector(".score").innerText = podiumArray[5]
            }
            gameOver.style.display = "flex"

            // clear sysChat area      
            sysChatDiv.innerHTML = ""
            // reset score board
            Object.keys(scoreBoard).forEach(k => {
                scoreBoard[k] = 0
            });
            playerBlocks.forEach( playerBlock => {
                playerBlock.querySelector(".score").innerText = 0
            })
            // display the podium and restart button
            isGameOver = true
            if (roomMaster === playerName) {
                if ( Object.keys(scoreBoard).length >= 2 ) {
                    restartBtn.style.display = "block"
                    gameOverWaitStart.style.display = "none"
                    gameOverWaitMember.style.display = "none"
                } else {
                    restartBtn.style.display = "none"
                    gameOverWaitStart.style.display = "none"
                    gameOverWaitMember.style.display = "flex"
                }
            } else {
                gameOverWaitStart.innerHTML =  "等待<b>" + roomMaster + "</b>開始"
                gameOverWaitStart.style.display = "flex"
            }
            
            break
    }
}

// 處理回合結束的reset
function reset() {
    // stop the previous clock
    stopCountingDown()
    // reset progress bar
    resetProgressBar()
    // clear canvas
    context.clearRect(0, 0, canvas.width, canvas.height);
    // reset the color options and current color
    colorOptions.forEach(option => option.classList.remove('selected'));
    currentColorDisplay.style.backgroundColor = "#000000"
    // reset stroke width
    strokeWidth = 5;
    strokeWidthInput.value = "5";
    // reset pen style
    penStyle = 1
    allPenStyle.forEach(pen => { if (pen.id == "pen") { pen.classList.add('selected') } else { pen.classList.remove('selected') } });
    
    // lock and reset the sysChat
    sysChatInput.value = ""
    sysChatInput.disabled = true
    overlay.style.display = "block"

    // reset score board status
    playerBlocks.forEach( playerBlock => {
        playerBlock.querySelector(".painter").style.display = "none"
        playerBlock.querySelector(".guessed").style.display = "none"
    })

    // reset guessed object
    guessed = {}
}

// 處理時間條

// 原本的方法，計時方面沒問題，但跟進度條動畫結合後有點問題
const MS_PER_SEC = 1000;
const ALARM_OFFSET = 10;  // in ms.
const MIN_INTERVAL = 100;  // in ms.
const progressFill = document.getElementById('progressFill');
let countdownTimerId;
var pointReward;

const stopCountingDown = () => {
    clearTimeout(countdownTimerId);
    resetProgressBar()
}

const ticker = (total, remaining) => {
    updateProgressBar(total, remaining);
};

function countdownTimer(duration, callback, tickCallback, interval = 1000) {
    console.log("enter countdownTimer")
    progressFill.style.width = '100%';
    interval = Math.max(MIN_INTERVAL, isNaN(interval) ? MIN_INTERVAL : interval);
    const alarmTime = performance.now() + duration * MS_PER_SEC;
    tick();
    function tick() {
        const timeTillAlarm = alarmTime - performance.now();
        
        if (timeTillAlarm < ALARM_OFFSET) {
            resetProgressBar()
            callback?.();
        } else {
            tickCallback(duration, timeTillAlarm); // total, remaining
            countdownTimerId = setTimeout(tick, Math.min(alarmTime - performance.now(), interval));
        }
    }
}

function resetProgressBar() {
    // Reset progress bar to empty and its opacity
    progressFill.style.opacity = "50%";
    progressFill.style.width = '0%';
}

function updateProgressBar(total, remaining) {
    // Update progress bar based on the remaining time
    const percentage = (remaining / (total * MS_PER_SEC)) * 100;
    pointReward = Math.round(percentage)
    progressFill.style.width = percentage + '%';
}

/* 第二種方法，進度條動畫很棒，但RO、RS同步和刷新進度條上面有點問題
    let start;
    let count = 0;
    let isCounting = false
    function stopCountingDown() {
        isCounting = true
    }

    function step(timestamp, duration, callback) {
        // "start" keep track of when the animation begins, so that subsequent frames can calculate 
        // the time elapsed since the start of the animation.
        if (isCounting) {
            isCounting = false
            resetProgressBar()
            return
        }

        if (start === undefined)
            start = timestamp;
        const elapsed = timestamp - start;

        progressBar.style.width = 100 - (100 / duration) * elapsed + "%";

        if (elapsed < duration) { // Stop the animation after the specified duration
            window.requestAnimationFrame(timestamp => step(timestamp, duration, callback));
        } else {
            resetProgressBar()
            callback?.()
        }
    }

    function resetProgressBar() {
        progressBar.style.display = "none"
        progressBar.style.width = "100%"
    }
*/


// 處理系統/公開聊天室
const sysChatInput = document.getElementById("sysChat").querySelector("input")
const publicChatInput = document.getElementById("publicChat").querySelector("input")

function padZero(value) {
    return value < 10 ? `0${value}` : value;
}

function timeStamp() {
    // Get current date and time
    const currentDate = new Date();

    // Get hours (in 24-hour format) and minutes
    const hours = currentDate.getHours();
    const minutes = currentDate.getMinutes();

    // Format the timestamp as a string
    const formattedTimestamp = `${padZero(hours)}:${padZero(minutes)}`;

    return `[${formattedTimestamp}]`
}

sysChatInput.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && sysChatInput.value.trim() != "") { // 防止只輸入""或是全部空格
        event.preventDefault(); // Prevent the default behavior (e.g., form submission)
        
        // Your custom logic here to modify the submission
        console.log('User submitted from sysChatInput:', sysChatInput.value);
        const stamp = timeStamp()
        send("sys", playerName + "```" + sysChatInput.value + "```" + stamp + "```" + pointReward)
        // Clear the input field if needed
        sysChatInput.value = "";
    }
});

publicChatInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter" && publicChatInput.value.trim() != "") { 
        event.preventDefault()
        console.log('User submitted from publicChatInput:', publicChatInput.value)
        const stamp = timeStamp()
        send("chat", playerName + "```" + publicChatInput.value + "```" + stamp)
        publicChatInput.value = ""
    }
})

// exit-cross
const exitCross = document.getElementById('exit-cross');
const confirmationDialog = document.getElementById('confirmation-dialog');
const confirmYes = document.getElementById('confirm-yes');
const confirmNo = document.getElementById('confirm-no');
const overlayDiv = document.createElement('div');
overlayDiv.classList.add('overlay-div');

exitCross.addEventListener('change', function () {
    if (exitCross.checked) {
        confirmationDialog.style.display = 'block';
        document.body.appendChild(overlayDiv);
        overlayDiv.style.display = 'block';
    } else {
        confirmationDialog.style.display = 'none';
        overlayDiv.style.display = 'none';
    }
});

confirmYes.addEventListener('click', function () {
    // Add your code for "Yes" confirmation here
    exitCross.checked = false; // Uncheck the checkbox
    confirmationDialog.style.display = 'none';
    overlayDiv.style.display = 'none';
    window.history.back(); // 回上一頁
});

confirmNo.addEventListener('click', function () {
    // Add your code for "No" confirmation here
    exitCross.checked = false; // Uncheck the checkbox
    confirmationDialog.style.display = 'none';
    overlayDiv.style.display = 'none';
});

overlayDiv.addEventListener('click', function () {
    exitCross.checked = false;
    confirmationDialog.style.display = 'none';
    overlayDiv.style.display = 'none';
});

// toggle switch
const toggleContainer = document.getElementById("toggle-container");
const privacyToggle = document.getElementById("privacyToggle");
const privacyStatus = document.getElementById("privacyStatus");

function togglePrivacy() {
    if (privacyToggle.checked) {
        // Room is private
        privacyStatus.innerText = "限邀請連結";
        privacyStatus.style.color = "#e74c3c"; // Red color for 'Private'
        var jsonObject = {"Type":"IN", "Payload":{"Content":"0"}};
        var jsonString = JSON.stringify(jsonObject);
        socket.send(jsonString)
    } else {
        // Room is public
        privacyStatus.innerText = "房號/邀請連結";
        privacyStatus.style.color = "#4CAF50"; // Green color for 'Public'
        var jsonObject = {"Type":"IN", "Payload":{"Content":"1"}};
        var jsonString = JSON.stringify(jsonObject);
        socket.send(jsonString)
    }
}

// exclamation mark
const exclamationMarkCheck = document.getElementById("exclamationMarkCheck")
const inviteLinkBox = document.getElementById("inviteLinkBox")
const copyInviteLink = document.getElementById("copyInviteLink")

exclamationMarkCheck.addEventListener('change', function () {
    if (exclamationMarkCheck.checked) {
        copyInviteLink.innerText = "複製"
        inviteLinkBox.style.display = 'flex';
        document.body.appendChild(overlayDiv);
        overlayDiv.style.display = 'block';
    } else {
        inviteLinkBox.style.display = 'none';
        overlayDiv.style.display = 'none';
    }
});

copyInviteLink.addEventListener("click", function() {
    var inviteLinkText = document.getElementById('inviteLink').innerText;

    // Create a temporary textarea element to hold the text
    var textarea = document.createElement('textarea');
    textarea.value = inviteLinkText;
    document.body.appendChild(textarea);

    // Select the text in the textarea
    textarea.select();
    textarea.setSelectionRange(0, textarea.value.length);

    // Copy the text to the clipboard
    document.execCommand('copy');

    // Remove the temporary textarea
    document.body.removeChild(textarea);

    copyInviteLink.innerText = "已複製"
    
})

overlayDiv.addEventListener('click', function () {
    exclamationMarkCheck.checked = false;
    inviteLinkBox.style.display = 'none';
    overlayDiv.style.display = 'none';
});