# HP P3015 Web Print Service (Extended Edition) 🖨️

A robust, driverless web print server written in Go. This tool provides a modern web interface that allows users to upload **Office documents and PDFs** directly to an **HP LaserJet P3015** (or similar legacy HP printers) without installing any drivers on the client device.

It is specifically engineered to solve common **"PDF Memory Overflow"** errors on older printers by preprocessing files on the server using **Ghostscript** and **LibreOffice**.

## ✨ Features

* **📄 Comprehensive File Support**: Now supports **Word (`.doc`, `.docx`), Excel (`.xls`, `.xlsx`), PowerPoint (`.ppt`, `.pptx`)**, in addition to PDF, PS, and TXT.
* **🧠 Smart Conversion Pipeline**:
    * **Office Files**: Automatically converted to PDF using headless LibreOffice.
    * **PDF Files**: Rasterized and optimized to **PostScript Level 2** using Ghostscript to prevent printer crashes.
* **🚀 Driverless Printing**: Users simply upload files via a web browser (Mobile & Desktop friendly).
* **🔒 HTTPS Bypass**: Uses `curl` internally to handle the printer's legacy self-signed certificates (`-k` mode) seamlessly.
* **💻 Cross-Platform**: Optimized for Linux (Ubuntu/Debian) servers, but fully compatible with macOS and Windows.

## 🛠️ Prerequisites

To ensure the conversion pipeline works, the host machine **must** have the following installed:

### 1. Linux (Ubuntu/Debian) - *Recommended*

You need Ghostscript for PDF processing, Curl for transmission, and LibreOffice for Word/Excel conversion. **Crucially, you must install fonts if printing non-English documents.**

```bash
sudo apt update

# Install core dependencies
sudo apt install ghostscript curl libreoffice -y

# Install Chinese/International fonts (Prevents squares/garbled text in Word files)
sudo apt install fonts-wqy-zenhei fonts-wqy-microhei -y
````

### 2\. macOS

⚠️ **Recommendation:** Use [Homebrew](https://brew.sh) for easy installation.

You need Ghostscript for PDF processing and LibreOffice for Word/Excel conversion. **The application automatically detects LibreOffice in `/Applications/LibreOffice.app`.**

```bash
# 1. Install dependencies
brew update
brew install curl ghostscript
brew install --cask libreoffice

# 2. Install Chinese/International fonts (Prevents squares/garbled text in Word files)
brew tap homebrew/cask-fonts
brew install --cask font-wqy-zenhei font-wqy-microhei
```

### 3\. Windows

1.  **Ghostscript**: Install [Ghostscript for Windows](https://www.ghostscript.com/releases/gsdnld.html).
2.  **LibreOffice**: Install [LibreOffice](https://www.libreoffice.org/download/download-libreoffice/). **Important:** During installation, ensure `soffice.exe` is added to your System PATH.
3.  **Curl**: Standard on Windows 10/11.

## 🚀 Quick Start

### 1\. Get the Application

**Option A: Download Binary**
Download the latest pre-compiled binary for your system from the [GitHub Releases](https://www.google.com/search?q=https://github.com/your-username/repo-name/releases) page.

**Option B: Build from Source**

```bash
# Initialize module
go mod init print-server

# Build binary
go build -o printer-service main.go
```

### 2\. Run the Server

By default, the server listens on port `8080` and targets the printer at `10.31.6.225`.

```bash
./printer-service
```

  * **Access:** Open your browser and navigate to `http://localhost:8080`
  * **Upload Limit:** The server accepts files up to **50MB**.

### 3\. Custom Configuration (Flags)

You can override the defaults using command-line flags:

```bash
# Example: Change port and printer IP
./printer-service -port="9000" -ip="192.168.1.50"

# Example: Specify custom Ghostscript path (if not in PATH)
./printer-service -gs="/usr/local/bin/gs"
```

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | Web server listening port |
| `-ip` | `10.31.6.225` | Target HP Printer IP address |
| `-gs` | `gs` (or `gswin64c`) | Path to the Ghostscript executable |

*(Note: LibreOffice command is auto-detected as `libreoffice` on Linux and `soffice` on Windows/macOS).*

## ⚙️ How It Works (The Pipeline)

```mermaid
    A[User Upload] -->|Office Doc| B(LibreOffice)
    A -->|PDF| C(Ghostscript)
    B -->|Convert to PDF| C
    C -->|Rasterize to PS Level 2| D{PostScript File}
    D -->|Curl -k| E[HP Printer API]
```

1.  **Ingest**: User uploads a file.
2.  **Detection & Conversion**:
      * **Office Files**: The server invokes `libreoffice --headless` to convert the document to **PDF**.
      * **PDF Files**: The server invokes `gs` to convert the PDF (or the one generated from Office) into **PostScript Level 2**. This "flattens" the file, reducing memory usage on the printer.
      * *GS Command used*: `-sDEVICE=ps2write -dLanguageLevel=2 -sPAPERSIZE=letter`.
3.  **Transmission**: The server uses `curl` to POST the final `.ps` file to the printer's internal API (`/hp/device/this.printservice`), bypassing SSL errors.

## ⚠️ Troubleshooting

**Q: Office files (Word/PPT) print with squares or garbage characters.**

  * **Cause**: The server is missing the necessary fonts.
  * **Fix (Linux)**: Install font packages: `sudo apt install fonts-wqy-zenhei`.
  * **Fix (macOS)**: Install fonts via Brew or verify System Fonts.
  * **Fix (Windows)**: Ensure the fonts used in the document are installed on the server machine.

**Q: Error "LibreOffice not found" or "exec: executable file not found".**

  * **Cause**: LibreOffice is not installed or not in the system PATH.
  * **Fix (macOS)**: Run the `sudo ln -s ...` command listed in the Prerequisites section.
  * **Fix (Windows)**: Add LibreOffice installation folder to your Environment Variables.

**Q: Error "PDF conversion failed".**

  * **Cause**: Ghostscript is missing.
  * **Fix**: Verify installation with `gs --version`.

**Q: Paper Size Issues (Letter vs A4).**

  * **Fix**: The code currently defaults to **Letter**. To change to A4, edit `main.go`:
    ```go
    // In func execGS, change:
    "-sPAPERSIZE=letter",
    // To:
    "-sPAPERSIZE=a4",
    ```

## 📄 License

GPLv3.
