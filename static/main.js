const fileListEl = document.querySelector("#fileList");

const btn = document.querySelector("#next");
const template = document.querySelector("#props");

function getprops() {
    return {
        startLimit: parseInt(template.getAttribute("data-limit")) || 0,
        limit: parseInt(template.getAttribute("data-limit")) || 0,
        counter: parseInt(template.getAttribute("data-counter")) || 0,
        offset: parseInt(template.getAttribute("data-offset")) || 0,
        keywords: template.getAttribute("data-keywords") || "",
        baseButtonURL: template.getAttribute("data-base-url") || "",
        order: null,
        orderAscending: null,
        replaceUrl: null,
    }
}

if (template) {

    let props = getprops();

    if (props.counter <= props.limit && btn) {
        btn.disabled = true; // Disable the button
    }


    props.offset += props.limit;
    props.baseButtonURL = `/append?keywords=${props.keywords}&limit=${props.limit}&offset=${props.offset}`;
    props.replaceUrl = `/search?keywords=${props.keywords}&limit=${props.limit}&offset=0`;

    changeBtnURL(props)
    updateList(props)
    updateTemplate(props)

    const sortParams = new Set(['name', 'size', 'modtime']);

    document.body.addEventListener('click', function (event) {

        const el = event.target.closest("[data-name]")
        const params = [el?.getAttribute("data-name"), el?.getAttribute("data-order")]


        const duplacetes = FindDuplcates(GetIds())
        if (duplacetes.length > 0)  {
            console.log(FindDuplcates(GetIds()))
        }


        if (event.target.id === 'next') {
            props.offset += props.limit;
            if (!props.order) {
                props.baseButtonURL = `/append?keywords=${props.keywords}&limit=${props.limit}&offset=${props.offset}`;
                props.replaceUrl = `/search?keywords=${props.keywords}&limit=${props.offset}&offset=0`;
            } else {
                props.baseButtonURL = `/append?keywords=${props.keywords}&limit=${props.limit}&offset=${props.offset}&order=${props.order}&ascending=${props.orderAscending ? 'true' : 'false'}`;
                props.replaceUrl = `/search?keywords=${props.keywords}&limit=${props.offset}&offset=0&order=${props.order}&ascending=${props.orderAscending ? 'true' : 'false'}`;
            }
            changeBtnURL(props)
            updateList(props)
            updateTemplate(props)
        } else if (sortParams.has(params[0])) {
            props.order = params[0]
            props.orderAscending = params[1]
            props.baseButtonURL = `/append?keywords=${props.keywords}&limit=${props.limit}&offset=${props.offset}&order=${params[0]}&ascending=${params[1] === 'ASC' ? 'true' : 'false'}`;
            props.replaceUrl = `/search?keywords=${props.keywords}&limit=${props.offset}&offset=0&order=${params[0]}&ascending=${params[1] === 'ASC' ? 'true' : 'false'}`;
            updateTemplate(props)
            changeBtnURL(props)
        }
        
        if (props.offset >= props.counter) {
            btn.disabled = true;
        } 
    });
}

let listContainer = document.getElementById("list");
if (listContainer) {
    document.getElementById("sort-name").addEventListener("click", (event) => handleSortClick(event))
    document.getElementById("sort-size").addEventListener("click", (event) => handleSortClick(event))
    document.getElementById("sort-date").addEventListener("click", (event) => handleSortClick(event))
}


function updateTemplate(props) {
    template.setAttribute("hx-get", props.baseButtonURL);
    template.setAttribute("data-offset", props.offset);
    template.setAttribute("data-order", props.order);
    htmx.process(template)
}

function changeBtnURL(props) {
    btn.setAttribute("hx-get", props.baseButtonURL);
    btn.setAttribute("hx-replace-url", props.replaceUrl);
    btn.setAttribute("data-offset", props.offset);
    htmx.process(btn)
}

function updateList(props) {
    const { keywords, offset, ...rest } = props
    const baseURL = `/append?keywords=${keywords}&limit=${offset}&offset=0`;
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
        const name = item.element.getAttribute("data-name")
        const ascdesc = item.element.getAttribute("data-order") === "ASC"
        item.element.setAttribute(
            'hx-get', baseURL + `&order=${name}&ascending=${ascdesc ? 'true' : 'false'}`
        )
        item.element.innerHTML = ascdesc ? "🠉" : "🠋"
        htmx.process(item.element)
    })
}






function handleSortClick(event) {

    listContainer = document.getElementById("list")
    listItems = Array.from(listContainer.querySelectorAll("li.item-list"))

    const target = event.target;
    const sortType = target.getAttribute("data-sort-type");
    const ascending = target.getAttribute("data-ascending") === "true";

    sortList(sortType, ascending);

    // Toggle the ascending/descending flag
    target.setAttribute("data-ascending", !ascending);
}


function FindDuplcates(arr) {
    const results = [];
    // pets.includes("cat") // true

    const seen = new Set();
    for (const value of arr) {
        if (seen.has(value)) {
            results.push(value);
        } else {
            seen.add(value);
        }
    }

    return results;
}

function GetIds() {
    const ids = [];
    const collection = document.querySelectorAll(".item-Id");
    collection.forEach(
        item => ids.push(item.textContent.replaceAll('\n', '').trim())
    )

    return ids;
}

