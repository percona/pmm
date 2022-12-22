function callback() {
    alert("script loaded successfully.");
    document.querySelector("body").setAttribute("style", "background-color: white");
}

function openDetails(e) {
    if (!e) {
        console.log("Not valid element");
    }

    let query = e.querySelector("*[role='cell']:first-child:first-child");
}