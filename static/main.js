document.addEventListener("DOMContentLoaded", function () {
    
    
    const buttonContainer = document.getElementById("button-container");
    const fileListEl = document.getElementById("fileList");


    const btn = document.querySelector("#next");


    const keywords = btn.getAttribute("data-keywords");
    const limit = parseInt(btn.getAttribute("data-limit"))
    const counter = parseInt(btn.getAttribute("data-counter"));
    let offset = parseInt(btn.getAttribute("data-offset"));

    
    if (counter <= limit && btn) {
        btn.disabled = true; // Disable the button
        btn.innerHTML = `There is ${counter} results`// Update button text
    }
    
    offset = offset + limit

    changeAtt()

    if (buttonContainer) {
        buttonContainer.addEventListener("click", function (event) {
            offset = offset + limit
            if (offset >= counter) {
                btn.disabled = true; // Disable the button
                return
            }
            changeAtt()
        })
    }
    
    
    function changeAtt() {
        let jsonURL = `/append?keywords=${keywords}&limit=${limit}&offset=${offset}`;
        // btn.innerHTML = jsonURL;
        btn.setAttribute("hx-get", jsonURL);
        btn.innerHTML = `There is ${counter} results` // Update button text
        htmx.process(btn)
    }


    let listContainer = document.getElementById("list"); // The parent container of the list items
    if (listContainer) {
        let listItems = Array.from(listContainer.querySelectorAll("li.item-list")); // Get all list items
        const sortObject = {
            name: true,
            size: true,
            date: true
        };
        let nameAsc = true; // Flag for ascending/descending order
        let sizeAsc = true; // Flag for ascending/descending order
        let dateAsc = true; // Flag for ascending/descending order

        document.getElementById("sort-size").addEventListener("click", (event) => handleSortClick(event))
        document.getElementById("sort-name").addEventListener("click", (event) => handleSortClick(event))
        document.getElementById("sort-date").addEventListener("click", (event) => handleSortClick(event))
    }

});
