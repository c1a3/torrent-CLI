const terminalOutput = document.getElementById('terminal-output');
const commandInput = document.getElementById('command-input');
const fileInput = document.getElementById('file-input');
const dropOverlay = document.getElementById('drop-overlay');

let ws = null;
let wsReconnectDelay = 1000;
let commandHistory = [];
let historyIndex = -1;
let torrents = {};
let isConnected = false;

const bannerASCII = [
    "",
    "░▀█▀░█▀█░█▀▄░█▀▄░█▀▀░█▀█░▀█▀░░░░░█▀▀░█░░░▀█▀",
    "░░█░░█░█░█▀▄░█▀▄░█▀▀░█░█░░█░░▄▄▄░█░░░█░░░░█░",
    "░░▀░░▀▀▀░▀░▀░▀░▀░▀▀▀░▀░▀░░▀░░░░░░▀▀▀░▀▀▀░▀▀▀",
];

const welcomeLines = [
    "",
    "  Type 'help' to see available commands.",
    "  Drag & drop a .torrent file anywhere to start downloading."
];

function printLine(text, className = '') {
    const el = document.createElement('div');
    el.className = 'output-line ' + className;
    el.textContent = text;
    terminalOutput.appendChild(el);
    scrollToBottom();
}

function scrollToBottom() {
    terminalOutput.scrollTop = terminalOutput.scrollHeight;
}

async function printBanner() {
    for (const line of bannerASCII) {
        printLine(line, 'success');
        await new Promise(r => setTimeout(r, 40));
    }
    
    for (const line of welcomeLines) {
        printLine(line, 'dim');
        await new Promise(r => setTimeout(r, 60));
    }
    printLine('', '');
}

function clearTerminal() {
    terminalOutput.innerHTML = '';
}

function renderProgressBar(progress, width = 30) {
    const filled = Math.floor((progress / 100) * width);
    const empty = width - filled;
    return '[' + '█'.repeat(Math.max(0, filled)) + '░'.repeat(Math.max(0, empty)) + '] ' + progress.toFixed(1) + '%';
}

function handleCommand(input) {
    const trimmed = input.trim();
    if (!trimmed) return;
    
    printLine('torrent@web:~$ ' + trimmed, 'command');
    
    commandHistory.push(trimmed);
    historyIndex = commandHistory.length;
    
    const parts = trimmed.match(/(?:[^\s"]+|"[^"]*")+/g) || [];
    const args = parts.map(arg => arg.replace(/^"|"$/g, ''));
    if (args.length === 0) return;
    
    const cmd = args[0].toLowerCase();
    const cmdArgs = args.slice(1);
    
    switch(cmd) {
        case 'help': showHelp(); break;
        case 'upload': triggerUpload(); break;
        case 'magnet': handleMagnet(cmdArgs.join(' ')); break;
        case 'status': case 'ls': showStatus(); break;
        case 'info': showInfo(cmdArgs[0]); break;
        case 'cancel': cancelTorrent(cmdArgs[0]); break;
        case 'download': downloadFile(cmdArgs[0], cmdArgs.slice(1).join(' ')); break;
        case 'files': showFiles(cmdArgs[0]); break;
        case 'clear': clearTerminal(); break;
        case 'system': showSystemInfo(); break;
        case 'about': showAbout(); break;
        default: printLine(`Command not found: ${cmd}. Type 'help' for available commands.`, 'error');
    }
}

function showHelp() {
    printLine('  COMMAND              DESCRIPTION', 'info');
    printLine('  ─────────────────────────────────────────────────────', 'dim');
    printLine('  help                 Show this help message');
    printLine('  upload               Upload a .torrent file');
    printLine('  magnet <uri>         Add a magnet link');
    printLine('  status / ls          List all torrents');
    printLine('  info <id>            Show detailed torrent info');
    printLine('  files <id>           List files in a torrent');
    printLine('  download <id> [path] Download a file from the torrent');
    printLine('  cancel <id>          Cancel an active download');
    printLine('  system               Show server system info');
    printLine('  clear                Clear the terminal');
    printLine('  about                About this application');
    printLine('');
}

async function triggerUpload() {
    fileInput.click();
}

async function uploadFile(file) {
    printLine(`Uploading ${file.name}...`, 'info');
    const formData = new FormData();
    formData.append('file', file);
    try {
        const res = await fetch('/api/upload', { method: 'POST', body: formData });
        const data = await res.json();
        if (data.success) {
            printLine(`✓ Torrent added: ${data.data.name} [${data.data.id}]`, 'success');
            printLine(`  Size: ${data.data.sizeFormatted}`, 'dim');
            printLine(`  Waiting for peers...`, 'dim');
        } else {
            printLine(`✗ Error: ${data.error}`, 'error');
        }
    } catch (e) {
        printLine(`✗ Upload failed: ${e.message}`, 'error');
    }
}

async function handleMagnet(uri) {
    if (!uri) {
        printLine('Usage: magnet <magnet-uri>', 'warning');
        return;
    }
    printLine('Adding magnet link...', 'info');
    try {
        const res = await fetch('/api/magnet', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ uri: uri })
        });
        const data = await res.json();
        if (data.success) {
            printLine(`✓ Magnet added [${data.data.id}]`, 'success');
            printLine(`  Resolving metadata from peers...`, 'dim');
        } else {
            printLine(`✗ Error: ${data.error}`, 'error');
        }
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

async function showStatus() {
    try {
        const res = await fetch('/api/torrents');
        const data = await res.json();
        if (data.success && data.data.length > 0) {
            printLine('');
            printLine('  ID        STATE          NAME                              PROGRESS', 'info');
            printLine('  ────────  ─────────────  ────────────────────────────────  ─────────────', 'dim');
            data.data.forEach(t => {
                const name = t.name.length > 30 ? t.name.substring(0, 27) + '...' : t.name.padEnd(30);
                const state = t.state.padEnd(13);
                const progress = t.state === 'downloading' ? renderProgressBar(t.progress, 15) : t.state;
                printLine(`  ${t.id.padEnd(8)}  ${state}  ${name}  ${progress}`);
            });
            printLine('');
        } else {
            printLine('No active torrents. Use "upload" or "magnet" to add one.', 'dim');
        }
    } catch (e) {
        printLine(`✗ Failed to fetch status: ${e.message}`, 'error');
    }
}

async function showInfo(id) {
    if (!id) { printLine('Usage: info <torrent-id>', 'warning'); return; }
    try {
        const res = await fetch(`/api/torrents/${id}`);
        const data = await res.json();
        if (data.success) {
            const t = data.data;
            printLine('');
            printLine(`  ╔══════════════════════════════════════════════╗`, 'info');
            printLine(`  ║  Torrent Details                             ║`, 'info');
            printLine(`  ╚══════════════════════════════════════════════╝`, 'info');
            printLine(`  ID:       ${t.id}`);
            printLine(`  Name:     ${t.name}`);
            printLine(`  Size:     ${t.sizeFormatted}`);
            printLine(`  State:    ${t.state}`);
            printLine(`  Progress: ${renderProgressBar(t.progress)}`);
            printLine(`  Speed:    ${t.speedFormatted}`);
            printLine(`  Peers:    ${t.peers}`);
            printLine(`  ETA:      ${t.eta}`);
            printLine(`  Added:    ${new Date(t.addedAt).toLocaleString()}`);
            if (t.error) printLine(`  Error:    ${t.error}`, 'error');
            printLine('');
        } else {
            printLine(`✗ Error: ${data.error}`, 'error');
        }
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

async function cancelTorrent(id) {
    if (!id) { printLine('Usage: cancel <torrent-id>', 'warning'); return; }
    try {
        const res = await fetch(`/api/torrents/${id}`, { method: 'DELETE' });
        const data = await res.json();
        if (data.success) {
            printLine(`✓ Torrent ${id} cancelled.`, 'success');
        } else {
            printLine(`✗ Error: ${data.error}`, 'error');
        }
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

async function showFiles(id) {
    if (!id) { printLine('Usage: files <torrent-id>', 'warning'); return; }
    try {
        const res = await fetch(`/api/torrents/${id}/files`);
        const data = await res.json();
        if (data.success && data.data.length > 0) {
            printLine('');
            printLine(`  Files in torrent ${id}:`, 'info');
            printLine('  ──────────────────────────────────────────────', 'dim');
            data.data.forEach((f, i) => {
                const progress = f.progress >= 100 ? '✓ DONE' : `${f.progress.toFixed(1)}%`;
                printLine(`  ${(i+1).toString().padStart(2)}. ${f.path}`);
                printLine(`      Size: ${f.sizeFormatted}  |  ${progress}`, 'dim');
            });
            printLine('');
        } else if (data.success) {
            printLine('No files found (metadata may still be loading).', 'dim');
        } else {
            printLine(`✗ Error: ${data.error}`, 'error');
        }
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

async function downloadFile(id, filePath) {
    if (!id) { printLine('Usage: download <torrent-id> [file-name]', 'warning'); return; }
    try {
        const res = await fetch(`/api/torrents/${id}/files`);
        const data = await res.json();
        if (!data.success) { printLine(`✗ Error: ${data.error}`, 'error'); return; }
        
        const files = data.data;
        if (files.length === 0) {
            printLine('No files available for download yet.', 'warning');
            return;
        }
        
        let targetPath = filePath;
        
        if (targetPath === '--all') {
            printLine(`Downloading all completed files as ZIP...`, 'info');
            const a = document.createElement('a');
            a.href = `/api/torrents/${id}/download_all`;
            a.download = `torrent_${id}.zip`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            printLine(`✓ ZIP download started in your browser.`, 'success');
            return;
        }

        if (!targetPath) {
            if (files.length === 1) {
                targetPath = files[0].path;
            } else {
                printLine(`Multiple files found. Specify a file name:`, 'warning');
                files.forEach((f, i) => {
                    const done = f.progress >= 100 ? '✓' : '…';
                    printLine(`  ${done} "${f.path}" (${f.sizeFormatted})`, 'dim');
                });
                printLine(`\nUsage: download ${id} "<specify-downloaded-file-name>" OR download ${id} --all`, 'info');
                return;
            }
        }
        
        // Validate the path exists in the torrent
        const fileExists = files.some(f => f.path === targetPath);
        if (!fileExists) {
            printLine(`✗ File not found in torrent: ${targetPath}`, 'error');
            printLine(`  Use "files ${id}" to see available files.`, 'dim');
            return;
        }
        
        printLine(`Downloading: ${targetPath}...`, 'info');
        const a = document.createElement('a');
        const encodedPath = targetPath.split('/').map(encodeURIComponent).join('/');
        a.href = `/api/torrents/${id}/download/${encodedPath}`;
        a.download = targetPath.split('/').pop();
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        printLine(`✓ Download started in your browser.`, 'success');
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

async function showSystemInfo() {
    try {
        const res = await fetch('/api/system');
        const data = await res.json();
        if (data.success) {
            const s = data.data;
            printLine('');
            printLine('  ╔══════════════════════════════════════════════╗', 'info');
            printLine('  ║  System Information                          ║', 'info');
            printLine('  ╚══════════════════════════════════════════════╝', 'info');
            printLine(`  Uptime:           ${s.uptime}`);
            printLine(`  Memory:           ${s.memoryUsedMB} MB / ${s.memoryTotalMB} MB`);
            printLine(`  Active Torrents:  ${s.activeTorrents}`);
            printLine(`  Completed:        ${s.completedTorrents}`);
            printLine(`  Bandwidth Used:   ${s.bandwidthFormatted}`);
            printLine('');
        }
    } catch (e) {
        printLine(`✗ Failed: ${e.message}`, 'error');
    }
}

function showAbout() {
    printLine('');
    printLine('  Torrent-CLI v1.2319', 'info');
    printLine('  ─────────────────────────────────────', 'dim');
    printLine('  Browser-based torrent downloader.');
    printLine('  Built with Go + anacrolix/torrent + vanilla JS.');
    printLine('  Deployed on Render.');
    printLine('');
    printLine('  ⚠  Files are ephemeral — download them before the', 'warning');
    printLine('     server sleeps.', 'warning');
    printLine('');
    printLine('  GitHub: github.com/c1a3/torrent-CLI', 'dim');
    printLine('');
}

function updateProgressLine(torrent) {
    let el = document.querySelector(`[data-progress-id="${torrent.id}"]`);
    const text = `  ↓ [${torrent.id}] ${renderProgressBar(torrent.progress, 25)} ${torrent.speedFormatted} | Peers: ${torrent.peers} | ETA: ${torrent.eta}`;
    
    if (!el) {
        el = document.createElement('div');
        el.className = 'output-line progress-line';
        el.setAttribute('data-progress-id', torrent.id);
        terminalOutput.appendChild(el);
    }
    el.textContent = text;
    scrollToBottom();
}

function removeProgressLine(id) {
    const el = document.querySelector(`[data-progress-id="${id}"]`);
    if (el) el.remove();
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
    
    ws.onopen = () => {
        isConnected = true;
        wsReconnectDelay = 1000;
        printLine('● WebSocket connected', 'dim');
    };
    
    ws.onclose = () => {
        isConnected = false;
        printLine('○ WebSocket disconnected. Reconnecting...', 'warning');
        setTimeout(connectWebSocket, wsReconnectDelay);
        wsReconnectDelay = Math.min(wsReconnectDelay * 2, 30000);
    };
    
    ws.onerror = (err) => {
        // Will trigger onclose
    };
    
    ws.onmessage = (event) => {
        const messages = event.data.split('\n');
        messages.forEach(msgStr => {
            if (!msgStr.trim()) return;
            try {
                const msg = JSON.parse(msgStr);
                handleWSMessage(msg);
            } catch (e) {
                console.error("WS Parse error", e);
            }
        });
    };
}

function handleWSMessage(msg) {
    switch (msg.type) {
        case 'torrent_info':
            const t = msg.payload;
            torrents[t.id] = t;
            printLine(`► Torrent ready: ${t.name} [${t.id}]`, 'success');
            printLine(`  Size: ${t.sizeFormatted} | Starting download...`, 'dim');
            break;
            
        case 'progress':
            const p = msg.payload;
            torrents[p.id] = p;
            updateProgressLine(p);
            break;
            
        case 'torrent_completed':
            const c = msg.payload;
            torrents[c.id] = c;
            removeProgressLine(c.id);
            printLine(`✓ Download complete: ${c.name} [${c.id}]`, 'success');
            printLine(`  Use "download ${c.id}" to save to your device.`, 'info');
            break;
            
        case 'torrent_cancelled':
            delete torrents[msg.payload.id];
            removeProgressLine(msg.payload.id);
            printLine(`✗ Torrent cancelled: ${msg.payload.id}`, 'warning');
            break;
            
        case 'torrent_error':
            const e = msg.payload;
            printLine(`✗ Error [${e.id}]: ${e.error}`, 'error');
            break;
            
        case 'warning':
            printLine(`⚠ Warning [${msg.payload.id}]: ${msg.payload.message}`, 'warning');
            break;
    }
}

// Drag and drop events
document.addEventListener('dragover', (e) => {
    e.preventDefault();
    dropOverlay.classList.add('active');
});

document.addEventListener('dragleave', (e) => {
    if (e.relatedTarget === null) dropOverlay.classList.remove('active');
});

document.addEventListener('drop', (e) => {
    e.preventDefault();
    dropOverlay.classList.remove('active');
    const files = Array.from(e.dataTransfer.files).filter(f => f.name.endsWith('.torrent'));
    if (files.length > 0) {
        files.forEach(f => uploadFile(f));
    } else {
        printLine('✗ Only .torrent files are accepted.', 'error');
    }
});

// Input handling
commandInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        const value = commandInput.value;
        commandInput.value = '';
        handleCommand(value);
    } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (historyIndex > 0) {
            historyIndex--;
            commandInput.value = commandHistory[historyIndex];
        }
    } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (historyIndex < commandHistory.length - 1) {
            historyIndex++;
            commandInput.value = commandHistory[historyIndex];
        } else {
            historyIndex = commandHistory.length;
            commandInput.value = '';
        }
    }
});

// Always focus the input
document.addEventListener('click', () => {
    // Only focus if not selecting text
    if (window.getSelection().toString() === '') {
        commandInput.focus();
    }
});

// File input change handler
fileInput.addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
        uploadFile(e.target.files[0]);
        fileInput.value = '';
    }
});

window.addEventListener('load', () => {
    printBanner();
    connectWebSocket();
    commandInput.focus();
});
