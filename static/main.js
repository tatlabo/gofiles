document.addEventListener("DOMContentLoaded", function () {
    
    const previewSectionEl = document.getElementById("preview-section");
    const fileListEl = document.getElementById("fileList");
    previewSectionEl.style.display = "none";


    const nextBtn = document.getElementById("next");
    const list = document.getElementById("list");
    const limit = parseInt("{{ .Params.Limit }}") || 10;
    let offset = limit
    const nameParam = "{{ .Params.Name }}";
    const buttonContainer = document.getElementById("button-container");
    const counter = parseInt("{{ .Counter }}") || 0;


    if (counter <= limit && nextBtn) {
        nextBtn.disabled = true; // Disable the button
        nextBtn.innerHTML = `There is ${counter} results`// Update button text
    }

    if (buttonContainer) {

        buttonContainer.addEventListener("click", function (event) {

            offset += limit

            // let like = `{{ .Params.Like }}` === `on` ? true : false
            // let dir = `{{ .Params.Dir }}` === `on` ? true : false


            let jsonURL = `/append?name=${nameParam}&limit=${limit}&offset=${offset}`


            if ("{{ .Params.Dir }}" === "on") {
                jsonURL += `&dir=on`;
            }

            if ("{{ .Params.Like }}" === "on") {
                jsonURL += `&like=on`;
            }

            if ("{{ .Params.Keywords }}" === "on") {
                jsonURL += `&keywords=on`;
            }

            console.log("jsonURL", jsonURL)

            nextBtn.innerHTML = `There is ${counter} results`;

            nextBtn.setAttribute("hx-get", jsonURL);

            if (offset >= counter) {
                nextBtn.disabled = true; // Disable the button
                nextBtn.innerHTML = `There is ${counter} results` // Update button text
            }
            htmx.process(nextBtn)

        })
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

})

