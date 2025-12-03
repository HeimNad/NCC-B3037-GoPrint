package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// --- Configuration ---
var (
	defaultPort      = "8080"
	defaultPrinterIP = "10.31.6.225"
	// "gs" command auto-detected based on OS
	gsCommand = "gs"
)

// HTML Template (English)
const htmlTemplateStr = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HP P3015 Print Service</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #f4f6f8; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .card { background: white; padding: 40px; border-radius: 16px; box-shadow: 0 10px 25px rgba(0,0,0,0.05); width: 100%; max-width: 420px; text-align: center; }
        h2 { color: #202124; margin-bottom: 8px; font-size: 24px; }
        .subtitle { color: #5f6368; font-size: 14px; margin-bottom: 30px; }
        .upload-area { border: 2px dashed #dadce0; padding: 40px 20px; border-radius: 8px; background: #fff; cursor: pointer; position: relative; transition: all 0.2s; }
        .upload-area:hover { border-color: #1a73e8; background: #f8faff; }
        input[type="file"] { position: absolute; top: 0; left: 0; width: 100%; height: 100%; opacity: 0; cursor: pointer; }
        .icon { font-size: 40px; color: #1a73e8; margin-bottom: 10px; display: block; }
        .btn { background: #1a73e8; color: white; border: none; padding: 12px 24px; width: 100%; border-radius: 6px; margin-top: 24px; font-size: 16px; font-weight: 500; cursor: pointer; transition: background 0.2s; }
        .btn:hover { background: #1557b0; box-shadow: 0 1px 3px rgba(0,0,0,0.3); }
        .btn:disabled { background: #dadce0; color: #fff; cursor: not-allowed; box-shadow: none; }
        .status { margin-top: 20px; padding: 12px; border-radius: 6px; font-size: 14px; line-height: 1.5; text-align: left; }
        .success { background: #e6f4ea; color: #137333; border: 1px solid #ceead6; }
        .error { background: #fce8e6; color: #c5221f; border: 1px solid #fad2cf; }
        .tech-info { font-size: 11px; color: #9aa0a6; margin-top: 30px; border-top: 1px solid #f1f3f4; padding-top: 15px; }
    </style>
    <script>
        function handleFileSelect(input) {
            const fileName = input.files[0].name;
            document.getElementById('fileLabel').innerText = fileName;
            document.getElementById('submitBtn').disabled = false;
        }
        function showLoading() {
            var btn = document.getElementById('submitBtn');
            var status = document.getElementById('statusMsg');
            btn.disabled = true;
            btn.innerText = "Processing & Sending...";
            if(status) status.style.display = 'none';
        }
    </script>
</head>
<body>
    <div class="card">
        <h2>🖨️ School Printer</h2>
        <p class="subtitle">Driverless printing for HP P3015</p>

        <form action="/upload" method="post" enctype="multipart/form-data" onsubmit="showLoading()">
            <div class="upload-area">
                <span class="icon">📄</span>
                <p id="fileLabel" style="margin:0; font-weight:500; color:#3c4043;">Click to select file</p>
                <p style="font-size: 12px; color: #5f6368; margin-top:5px;">Supports: PDF, PS, TXT</p>
                <input type="file" name="file" required accept=".pdf,.ps,.prn,.txt" onchange="handleFileSelect(this)">
            </div>
            <button type="submit" id="submitBtn" class="btn" disabled>Print Now</button>
        </form>

        {{if .Message}}
        <div id="statusMsg" class="status {{.StatusClass}}">
            {{.Message}}
        </div>
        {{end}}

        <div class="tech-info">
            System: Ubuntu 22.04 LTS | Engine: Go + Ghostscript<br>
            Note: PDF files are automatically optimized to prevent printer memory errors.
        </div>
    </div>
</body>
</html>
`

func main() {
	// Auto-detect OS for Ghostscript command
	if runtime.GOOS == "windows" {
		gsCommand = "gswin64c"
	} else {
		gsCommand = "gs" // Linux standard command
	}

	port := flag.String("port", defaultPort, "Server Port")
	printerIP := flag.String("ip", defaultPrinterIP, "Printer IP Address")
	gsPath := flag.String("gs", gsCommand, "Path to Ghostscript executable")
	flag.Parse()

	gsCommand = *gsPath
	targetURL := fmt.Sprintf("https://%s/hp/device/this.printservice?printThis", *printerIP)

	// Check GS availability
	if !checkGS() {
		fmt.Println("⚠️  WARNING: Ghostscript ('gs') not found.")
		fmt.Println("   PDF conversion will fail. Please install it via: sudo apt install ghostscript")
	} else {
		fmt.Printf("✅ Ghostscript found: %s\n", gsCommand)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			render(w, "", "")
			return
		}
		handleUpload(w, r, targetURL)
	})

	fmt.Printf("🚀 Server started on port %s\n", *port)
	fmt.Printf("🖨️  Target Printer: %s\n", *printerIP)
	http.ListenAndServe(":"+*port, nil)
}

func handleUpload(w http.ResponseWriter, r *http.Request, targetURL string) {
	// 1. Limit upload size (e.g., 20MB)
	r.ParseMultipartForm(20 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		render(w, "Error: No file uploaded.", "error")
		return
	}
	defer file.Close()

	// Save uploaded file to temp
	tempInput, err := os.CreateTemp("", "print_upload_*.tmp")
	if err != nil {
		render(w, "Server Error: Cannot create temp file.", "error")
		return
	}
	defer os.Remove(tempInput.Name())
	defer tempInput.Close()

	_, err = io.Copy(tempInput, file)
	if err != nil {
		render(w, "Server Error: File save failed.", "error")
		return
	}
	tempInput.Close()

	finalFilePath := tempInput.Name()
	statusMsg := fmt.Sprintf("✅ Success! '%s' sent to printer.", header.Filename)

	// 2. Logic: If PDF, convert via Ghostscript
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".pdf" {
		fmt.Printf("🔄 Converting PDF: %s ...\n", header.Filename)

		tempOutput := tempInput.Name() + ".ps"

		// Ghostscript arguments for Linux
		err := execGS(tempInput.Name(), tempOutput)
		if err != nil {
			fmt.Printf("❌ Conversion Failed: %v\n", err)
			render(w, "Error: PDF conversion failed. The server might be missing Ghostscript.", "error")
			return
		}

		finalFilePath = tempOutput
		defer os.Remove(tempOutput)
		statusMsg += " (Optimized via Ghostscript)"
		fmt.Println("✅ Conversion done.")
	}

	// 3. Send to Printer
	err = sendToPrinter(finalFilePath, targetURL)
	if err != nil {
		fmt.Printf("❌ Send Failed: %v\n", err)
		render(w, fmt.Sprintf("Printer Error: %v. Check if printer is online.", err), "error")
		return
	}

	render(w, statusMsg, "success")
}

func execGS(inputPath, outputPath string) error {
	// Recommended settings for HP P3015 compatibility
	cmd := exec.Command(gsCommand,
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		"-sDEVICE=ps2write", // PS Level 2 is safer for old printers
		"-dLanguageLevel=2",
		"-sPAPERSIZE=letter", // Default to letter, change to 'A4' if you need
		"-sOutputFile="+outputPath,
		inputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("cmd error: %v, output: %s", err, stderr.String())
	}
	return nil
}

func sendToPrinter(filePath, url string) error {
	fmt.Println("⚡ Switching to Curl mode due to certificate error...")

	// curl command:
	// -k: ignore certificate errors
	// -F: form data
	// LocalFile=@...: file uri
	// htxtRedirect=...: form field
	cmd := exec.Command("curl",
		"-k",
		"-F", fmt.Sprintf("LocalFile=@%s", filePath),
		"-F", "htxtRedirect=hp.Print",
		url,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("curl failed: %v | Log: %s", err, stderr.String())
	}

	return nil
}

func render(w http.ResponseWriter, msg, statusClass string) {
	t, _ := template.New("page").Parse(htmlTemplateStr)
	t.Execute(w, struct {
		Message     string
		StatusClass string
	}{msg, statusClass})
}

func checkGS() bool {
	_, err := exec.LookPath(gsCommand)
	return err == nil
}
