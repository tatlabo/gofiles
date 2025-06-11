
// On page load, set theme from localStorage if available
document.addEventListener("DOMContentLoaded", function () {
    const savedTheme = localStorage.getItem("theme");
    const themeForm = document.getElementById("theme");


    if (savedTheme && themeForm) {
        document.documentElement.setAttribute("data-theme", savedTheme);
        // Set the radio button as checked
        const radio = themeForm.querySelector(`input[name="theme"][value="${savedTheme}"]`);
        if (radio) radio.checked = true;
    }

    if (themeForm) {
        themeForm.oninput = function (e) {
            const value = e.target.value;
            document.documentElement.setAttribute("data-theme", value);
            localStorage.setItem("theme", value);
        };
    }
});

// theme.oninput = e => {
//     document.firstElementChild.setAttribute('data-theme', e.target.value)
// }

const themeToggleBtn = document.getElementById("theme-toggle");
const sun = "☀";
const moon = "☽"

document.addEventListener("DOMContentLoaded", () => {
    const savedTheme = localStorage.getItem("theme") || "light"; // Default to light theme if not set
    setTheme(savedTheme); // Set the initial theme

    if (themeToggleBtn) {
        if (savedTheme === "dark" && themeToggleBtn !== null) {
            themeToggleBtn.textContent = sun;
        } else {
            themeToggleBtn.textContent = moon;
        }
    }
})

if (themeToggleBtn) {
    

    themeToggleBtn.addEventListener("click", (e) => {

        const root = document.firstElementChild;
        let currentTheme = root.getAttribute("data-theme");

        if (currentTheme === "dark") {
            root.setAttribute('data-theme', 'light')
            themeToggleBtn.textContent = moon
        } else {
            root.setAttribute('data-theme', 'dark')
            themeToggleBtn.textContent = sun
        }

        currentTheme = currentTheme === "light" ? "dark" : "light";
        localStorage.setItem("theme", currentTheme);

    })
}







function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
}


function sortList(parameter, ascending = true) {
    listItems.sort((a, b) => {
        const aValue = a.dataset[parameter]; // Get the data attribute value
        const bValue = b.dataset[parameter];

        switch (parameter) {
            case "name":
                return ascending ? aValue.localeCompare(bValue) : bValue.localeCompare(aValue);
            case "size":
                return ascending ? parseInt(aValue) - parseInt(bValue) : parseInt(bValue) - parseInt(aValue);
            case "moddate":
                let a = aValue.replace(/\s+[A-Z]+$/, "")
                let b = bValue.replace(/\s+[A-Z]+$/, "")
                return ascending ? new Date(a) - new Date(b) : new Date(b) - new Date(a);
            default:
                return 0;
        }
        const aParsed = parameter === "size" || parameter === "moddate" ? parseFloat(aValue) : aValue.toLowerCase();
        const bParsed = parameter === "size" || parameter === "moddate" ? parseFloat(bValue) : bValue.toLowerCase();

        if (aParsed < bParsed) return ascending ? -1 : 1;
        if (aParsed > bParsed) return ascending ? 1 : -1;
        return 0;
    });

    // Use a DocumentFragment to append all sorted items at once
    const fragment = document.createDocumentFragment();
    listItems.forEach(item => fragment.appendChild(item)); // Add sorted items to the fragment

    // Clear the current list and append the fragment
    listContainer.innerHTML = ""; // Clear the current list
    listContainer.appendChild(fragment); // Append all items at once
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


function hidePreview() {
    previewSectionEl.style.display = "none";
}

document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" || event.key === "Esc") {
        if (typeof previewSectionEl !== "undefined" && previewSectionEl.style.display !== "none") {
            hidePreview();
        }
    }
});

function showPreview() {
    if (previewSectionEl.style.display === "none") {
        previewSectionEl.style.display = "";
    }

    if (previewSectionEl.style.display !== "none") {
        const hidePreviewBtn = document.getElementById("hidePreview");
        hidePreviewBtn.addEventListener("click", hidePreview);

    }
}
