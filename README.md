# HP P3015 Web Print Service 🖨️

A lightweight, driverless web print server written in Go. This tool provides a modern web interface that allows users to upload documents directly to an **HP LaserJet P3015** (or similar legacy HP printers) without installing any drivers on the client device.

It is specifically engineered to solve common **"PDF Memory Overflow"** or interpreter errors on older printers by preprocessing PDFs on the server using **Ghostscript**.

## ✨ Features

  * **Driverless Printing**: Users simply upload files via a web browser (works on Mobile & Desktop).
  * **Smart PDF Optimization**: Automatically detects PDF files and uses Ghostscript to convert them into **PostScript Level 2**. This prevents the printer from crashing on complex modern PDFs.
  * **Modern UI**: Single-file implementation with an embedded, responsive Material Design HTML template.
  * **HTTPS Bypass**: Uses `curl` internally to handle the printer's self-signed certificates (`-k` mode) seamlessly.
  * **Cross-Platform Server**: Designed for Linux (Ubuntu/Debian) but supports Windows.

## 🛠️ Prerequisites

To ensure the PDF conversion and file transmission work correctly, the host machine must have the following installed:

1.  **Ghostscript (`gs`)**: Required for rasterizing PDFs to PostScript.
2.  **Curl**: Required for sending the data stream to the printer while ignoring SSL errors.

### Linux (Ubuntu/Debian)

```bash
sudo apt update
sudo apt install ghostscript curl
```

### Windows

  * Install [Ghostscript for Windows](https://www.ghostscript.com/releases/gsdnld.html).
  * Ensure `curl` is in your system PATH (Standard on Windows 10/11).

## 🚀 Quick Start

### 1\. Build the Application

```bash
# Initialize module (if not already done)
go mod init print-server

# Build binary
go build -o printer-service main.go
```

### 2\. Run the Server

By default, the server listens on port `8080` and targets the printer at `10.31.6.225`.

```bash
./printer-service
```

*Open your browser and navigate to:* `http://localhost:8080`

### 3\. Custom Configuration (Flags)

You can override the defaults using command-line flags:

```bash
# Example: Change port and printer IP
./printer-service -port="9000" -ip="192.168.1.50"

# Example: Specify custom Ghostscript path
./printer-service -gs="/usr/local/bin/gs"
```

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | Web server listening port |
| `-ip` | `10.31.6.225` | Target HP Printer IP address |
| `-gs` | `gs` (or `gswin64c`) | Path to the Ghostscript executable |

## ⚙️ How It Works

1.  **Upload**: User selects a file (`.pdf`, `.ps`, `.txt`) via the web interface.
2.  **Processing**:
      * **If PDF**: The server invokes `gs`. It converts the PDF to **PostScript Level 2** (safer for old hardware) and forces the paper size to **Letter**.
      * *GS Command used*: `-sDEVICE=ps2write -dLanguageLevel=2 -sPAPERSIZE=letter`.
      * **If other**: The file is kept as-is.
3.  **Transmission**: The server uses `curl` to POST the file to the printer's internal API (`/hp/device/this.printservice`), bypassing SSL certificate validation.
4.  **Feedback**: The user receives a success or error message immediately.

## ⚠️ Troubleshooting

**Q: Error "PDF conversion failed"**

  * **Cause**: Ghostscript is not installed or not found in the system PATH.
  * **Fix**: Verify installation with `gs --version`. On Windows, you might need to use the flag `-gs "gswin64c.exe"`.

**Q: Error "Printer Error... Check if printer is online"**

  * **Cause**: The server cannot reach the printer IP, or the printer's web server is disabled.
  * **Fix**:
    1.  Ping the printer IP from the server.
    2.  Ensure the printer has HTTP/HTTPS services enabled in its networking config.
    3.  Verify the `targetURL` in the code matches your specific HP model's API.

**Q: Wrong Paper Size (Letter vs A4)**

  * **Fix**: The code defaults to Letter. To change this to Letter, edit `main.go`:
    ```go
    // In func execGS, change:
    "-sPAPERSIZE=letter",
    // To:
    "-sPAPERSIZE=a4",
    ```

## 📝 Future Improvements

  * [ ] Add support for image printing (JPG/PNG to PostScript).

## 📄 License

GPLv3 License.