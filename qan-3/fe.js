let ws = new WebSocket("ws://localhost:8080");
ws.onmessage = function (res) { 
    let response = JSON.parse(res.data);

    if (response.Error != "") {
        console.log(response.Error);
        return;
    }

    if (response.Target != "") { 
        let e = document.querySelector(response.Target)
        e ? e.innerHTML = response.HTML : console.log("Element " + response.Target + " doesnt exists");
    }

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
    request("get", "test");
}

function request(kind, data) {
    if (!data)
        console.log("No data provied");

    let req = JSON.stringify({
        Kind: kind,
        Data: data
    });

    ws.send(req);
}