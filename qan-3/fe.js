let ws = new WebSocket("ws://localhost:8080");
ws.onmessage = function (x) { document.querySelector("section.panel-container").innerHTML = x.data; if (done) { done(); } }
ws.onopen = () => { ws.send("test") }