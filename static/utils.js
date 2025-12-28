
// On page load, set theme from localStorage if available

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

const previewSectionEl = document.getElementById("preview-section");

function hidePreview() {
    previewSectionEl.style.display = "none";
}

function deletePreview() {
    const p = document.getElementById("preview");
    while (p.firstChild) {
        p.removeChild(p.firstChild);
}
    previewSectionEl.style.display = "none";
}

function showPreview() {
    previewSectionEl.style.display = "block";
}

function showImage(e) {
    console.log("Showing image:", e);

    const image = document.createElement("img");
    image.src = e;
    image.style.maxWidth = "100%";
    image.style.height = "auto";

    const previewImg = document.getElementById("preview-image");
    previewImg.appendChild(image);
    previewImg.style.display = "block";


}

if (previewSectionEl) {
        
    document.addEventListener("keydown", function (event) {
            if (event.key === "Escape" || event.key === "Esc") {
                    if (typeof previewSectionEl !== "undefined" && previewSectionEl.style.display !== "none") {
                            deletePreview();
                        }
                    }
                });
                
}

            

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
    });

    // Use a DocumentFragment to append all sorted items at once
    const fragment = document.createDocumentFragment();
    listItems.forEach(item => fragment.appendChild(item)); // Add sorted items to the fragment

    // Clear the current list and append the fragment
    listContainer.innerHTML = ""; // Clear the current list
    listContainer.appendChild(fragment); // Append all items at once
}








