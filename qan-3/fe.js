let ws = new WebSocket("ws://localhost:8080");
ws.onmessage = function (res) { 
    let response = JSON.parse(res.data);

    if (response.Error != "") {
        console.log(response.Error);
        return;
    }

    let e = document.querySelector(response.Target)
    if (!e) {
        console.log("Element " + response.Target + " doesnt exists");
        return;
    } 
    e.innerHTML = response.HTML;

    if (response.Script != "") { 
        let script = document.createElement('script');
        //script.setAttribute("id", "responseScript");
        script.innerText = response.Script;
        document.head.appendChild(script);

        if (typeof callback === "function")
            callback();
    }
}
ws.onopen = () => {
    document.querySelector(".overview-filters").innerText = "loading...";
    document.querySelector(".query-analytics-data").innerText = "loading...";
    setTimeout(ws.send("filter"), 2000);
    setTimeout(ws.send("content"), 3000);
}