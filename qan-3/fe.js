let ws = new WebSocket("ws://localhost:8080");
ws.onmessage = function (res) { 
    let response = JSON.parse(res.data);

    if (response.Error != "") {
        console.log(response.Error);
    }

    document.querySelector("section.panel-container").innerHTML = response.HTML;

    if (response.Script != "") { 
        let script = document.createElement('script');
        //script.setAttribute("id", "responseScript");
        script.innerText = response.Script;
        document.head.appendChild(script);

        if (typeof callback === "function") { 
            callback();
        }
    }
}
ws.onopen = () => { ws.send("test"); }