function callback() {
    alert("script loaded successfully.");
}

function openDetails(e) {
    if (!e) {
        console.log("Not valid element");
    }

    let query = e.querySelector("*[role='cell']:nth-child(2)");
}