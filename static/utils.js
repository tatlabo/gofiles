
// On page load, set theme from localStorage if available



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

// theme.oninput = e => {
//     document.firstElementChild.setAttribute('data-theme', e.target.value)
// }

const themeToggleBtn = document.getElementById("theme-toggle");
const sun = "☀";
const moon = "☽"


function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
}


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












