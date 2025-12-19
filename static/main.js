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




const  baseURL = `/append?keywords=${keywords}&limit=${limit}&offset=${offset}`;
const sortButtons = [
    {
        element: document.querySelector(".ascending.name"),
        ascending: true
    },
    {
        element: document.querySelector(".descending.name"),
    },
]

// const _ = sortButtons.map(item => {
//     const ascdesc = item.ascending ? "true" : "false"
//     const name = item.element.getAttribute("data-name")
//     item.element.setAttribute("hx-get", baseURL + `&order=${item.type}&ascending=${item.ascending}&order=${name}&ascending=${ascdesc}`)
//     htmx.process(item.element)
// })

// const nameAsc = document.querySelector(".ascending.name");
// let desc = "desc"
// nameAsc.setAttribute("hx-get", baseURL + sufix)
// htmx.process(nameAsc)

// const nameDesc = document.querySelector(".descending.name");
// let sufix = `&order=name&ascending=false`
// nameAsc.setAttribute("hx-get", baseURL + sufix)
// htmx.process(nameAsc)
