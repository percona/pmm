function callback() {
    alert("all images and submits to Percona");
    document.querySelectorAll("img").forEach((e) => { e.setAttribute("src", "https://pmmdemo.percona.com/graph/public/img/percona-logo.svg")});
    document.querySelectorAll("input[type=submit]").forEach((e) => { e.value = "Percona"});
}