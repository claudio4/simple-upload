import * as tus from 'tus-js-client'

const dropZone = document.getElementById("drop-zone");
const fileInput = document.getElementById("file-input");
const progressBar = document.getElementById("progress-bar");
const progressText = document.getElementById("progress-text");
const statusText = document.getElementById("status");

const UPLOAD_URL = "/files/";

// Click to open file selector
dropZone.addEventListener("click", () => fileInput.click());

// File selection
fileInput.addEventListener("change", (event) => {
    if (event.target.files.length > 0) {
        uploadFile(event.target.files[0]);
    }
});

// Drag & Drop events
dropZone.addEventListener("dragover", (event) => {
    event.preventDefault();
    dropZone.classList.add("highlight");
});

dropZone.addEventListener("dragleave", (event) => {
    dropZone.classList.remove("highlight");
});

dropZone.addEventListener("drop", (event) => {
    event.preventDefault();
    dropZone.classList.remove("highlight");
    if (event.dataTransfer.files.length > 0) {
        uploadFile(event.dataTransfer.files[0]);
    }
});

function uploadFile(file) {
    progressBar.style.width = "0%";
    progressText.textContent = "0%";
    statusText.textContent = "";
    statusText.classList.remove("error", "success");

    const upload = new tus.Upload(file, {
        endpoint: UPLOAD_URL,
        retryDelays: [0, 1000, 3000, 5000],
        metadata: {
            filename: file.name,
            filetype: file.type,
        },
        onError: function (error) {
            console.error("Upload failed:", error);
            statusText.textContent = "Upload failed. Try again.";
            statusText.classList.add("error");
        },
        onProgress: function (bytesUploaded, bytesTotal) {
            const percentage = ((bytesUploaded / bytesTotal) * 100).toFixed(2);
            progressBar.style.width = percentage + "%";
            progressText.textContent = percentage + "%";
        },
        onSuccess: function () {
            console.log("Upload finished:", upload.url);
            progressBar.style.width = "100%";
            progressText.textContent = "100%";
            statusText.textContent = "Upload successful!";
            statusText.classList.add("success");
        },
    });

    // Check if there are any previous uploads to continue.
    upload.findPreviousUploads().then(function (previousUploads) {
        if (previousUploads.length) {
            upload.resumeFromPreviousUpload(previousUploads[0]);
        }
        upload.start();
    });
}
