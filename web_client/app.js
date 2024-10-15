let mediaRecorder;
let recordedChunks = [];

// WebSocket for real-time updates
const wsProtocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
const ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws`);

ws.onmessage = function(event) {
    console.log("WebSocket message:", event.data);
    showAlert(event.data, 'info');
};

// Utility function to show alerts
function showAlert(message, type) {
    const alertPlaceholder = document.createElement('div');
    alertPlaceholder.className = `alert alert-${type} alert-dismissible fade show`;
    alertPlaceholder.role = 'alert';
    alertPlaceholder.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;
    document.body.prepend(alertPlaceholder);
    setTimeout(() => {
        alertPlaceholder.remove();
    }, 5000);
}

// Start Stream Button
document.getElementById('start-stream').addEventListener('click', function() {
    fetch('/start-stream')
        .then(response => response.json())
        .then(data => {
            showAlert(data.status, 'success');
        })
        .catch(err => {
            console.error(err);
            showAlert('Error starting stream', 'danger');
        });
});

// Stop Stream Button
document.getElementById('stop-stream').addEventListener('click', function() {
    fetch('/stop-stream')
        .then(response => response.json())
        .then(data => {
            showAlert(data.status, 'success');
        })
        .catch(err => {
            console.error(err);
            showAlert('Error stopping stream', 'danger');
        });
});

// Start Recording Button
document.getElementById('start-recording').addEventListener('click', function() {
    const videoElement = document.getElementById('video-player');
    const stream = videoElement.captureStream();
    mediaRecorder = new MediaRecorder(stream);

    mediaRecorder.ondataavailable = function(event) {
        if (event.data.size > 0) {
            recordedChunks.push(event.data);
        }
    };

    mediaRecorder.onstop = function() {
        const blob = new Blob(recordedChunks, { type: 'video/mp4' });
        const url = URL.createObjectURL(blob);

        const downloadLink = document.getElementById('download-link');
        downloadLink.href = url;
        downloadLink.style.display = 'inline-block';
        recordedChunks = [];

        showAlert('Recording stopped. You can download the video.', 'success');
    };

    mediaRecorder.start();
    document.getElementById('start-recording').disabled = true;
    document.getElementById('stop-recording').disabled = false;
    showAlert('Recording started', 'success');
});

// Stop Recording Button
document.getElementById('stop-recording').addEventListener('click', function() {
    mediaRecorder.stop();
    document.getElementById('start-recording').disabled = false;
    document.getElementById('stop-recording').disabled = true;
});

// Fetch and display video list
function fetchVideoList() {
    fetch('/list-videos')
        .then(response => response.json())
        .then(data => {
            const videoList = document.getElementById('video-list');
            videoList.innerHTML = '';
            if (data.videos.length === 0) {
                const li = document.createElement('li');
                li.className = 'list-group-item text-center';
                li.textContent = 'No recordings available.';
                videoList.appendChild(li);
                return;
            }
            data.videos.forEach(video => {
                const li = document.createElement('li');
                li.className = 'list-group-item';
                const a = document.createElement('a');
                a.href = `/videos/${video}`;
                a.textContent = video;
                a.target = '_blank';
                li.appendChild(a);
                videoList.appendChild(li);
            });
        })
        .catch(err => {
            console.error(err);
            showAlert('Error fetching video list', 'danger');
        });
}

// Initial fetch
fetchVideoList();

// Refresh video list every minute
setInterval(fetchVideoList, 60000);