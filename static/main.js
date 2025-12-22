const buttonContainer = document.querySelector("div.button-container");
const fileListEl = document.querySelector("#fileList");

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
updateList(offset)
// if (btn !== null) {


const sizeAsc = document.querySelector(".ascending.size");
const sizeDesc = document.querySelector(".descending.size");
const dateAsc = document.querySelector(".ascending.modtime");
const dateDesc = document.querySelector(".descending.modtime");


if (buttonContainer) {
    buttonContainer.addEventListener("click", function (event) {
        offset = offset + limit
        if (offset >= counter) {
            btn.disabled = true; // Disable the button
            return
        }
        changeAtt()
        updateList(offset)
    })
}


function changeAtt() {
    let jsonURL = `/append?keywords=${keywords}&limit=${limit}&offset=${offset}`;
    btn.setAttribute("hx-get", jsonURL);
    btn.innerHTML = `There is ${counter} results`
    htmx.process(btn)
}


let listContainer = document.getElementById("list");
if (listContainer) {
    // let listItems = Array.from(listContainer.querySelectorAll("li.item-list"));
    // const sortObject = {
    //     name: true,
    //     size: true,
    //     date: true
    // };
    // let nameAsc = true;
    // let sizeAsc = true;
    // let dateAsc = true;

    document.getElementById("sort-name").addEventListener("click", (event) => handleSortClick(event))
    document.getElementById("sort-size").addEventListener("click", (event) => handleSortClick(event))
    document.getElementById("sort-date").addEventListener("click", (event) => handleSortClick(event))
}


function handleSortClick(event) {

    listContainer = document.getElementById("list")
    listItems = Array.from(listContainer.querySelectorAll("li.item-list"))

    const target = event.target;
    const sortType = target.getAttribute("data-sort-type");
    const ascending = target.getAttribute("data-ascending") === "true";

    console.log("sortType", sortType)

    // Sort the list based on the clicked header
    sortList(sortType, ascending);

    // Toggle the ascending/descending flag
    target.setAttribute("data-ascending", !ascending);
}


function updateList(offset) {
    const  baseURL = `/append?keywords=${keywords}&limit=${offset}&offset=0`;
    const sortButtons = [
        {
            element: document.querySelector(".ascending.name"),
            ascending: true
        },
        {
            element: document.querySelector(".descending.name"),
        },
        {
            element: document.querySelector(".ascending.size"),
            ascending: true
        },
        {
            element: document.querySelector(".descending.size"),
        },
        {
            element: document.querySelector(".ascending.modtime"),
            ascending: true
        },
        {
            element: document.querySelector(".descending.modtime"),
        },
        
    ]
    
    const _ = sortButtons.map(item => {
        let name = item.element.getAttribute("data-name")
        const ascdesc = item.ascending ? "true" : "false"
        item.element.setAttribute(
            'hx-get', baseURL + `&order=${name}&ascending=${ascdesc}`
        )
        htmx.process(item.element)
    })
}



