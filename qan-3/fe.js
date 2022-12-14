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
    setTimeout(() => { request("filter"); }, 2000);
    setTimeout(() => { request("content"); }, 3000);
}

function request(type, data) {
    if (!data) {
        console.log("No data provied");
    }

    let req = JSON.stringify({
        Type: type,
        Data: data
    });

    ws.send(req);
}