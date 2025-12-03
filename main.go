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
	gsCommand        = "gs"
	officeCommand    = "libreoffice"
)

// HTML Template
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
                <p style="font-size: 12px; color: #5f6368; margin-top:5px;">Supports: Word, Excel, PPT, PDF</p>
                <input type="file" name="file" required accept=".pdf,.ps,.prn,.txt,.doc,.docx,.xls,.xlsx,.ppt,.pptx" onchange="handleFileSelect(this)">
            </div>
            <button type="submit" id="submitBtn" class="btn" disabled>Print Now</button>
        </form>

        {{if .Message}}
        <div id="statusMsg" class="status {{.StatusClass}}">
            {{.Message}}
        </div>
        {{end}}

        <div class="tech-info">
            Engine: Go + Ghostscript + LibreOffice<br>
            Note: Office files are converted to PDF, then optimized for printer memory.
        </div>
    </div>
</body>
</html>
`

func main() {
	switch runtime.GOOS {
	case "windows":
		gsCommand = "gswin64c"
		// On Windows, assume LibreOffice is installed as "soffice"
		officeCommand = "soffice"
	case "darwin":
		gsCommand = "gs"
		officeCommand = "/Applications/LibreOffice.app/Contents/MacOS/soffice"
	default:
		gsCommand = "gs"
		officeCommand = "libreoffice"
	}

	port := flag.String("port", defaultPort, "Server Port")
	printerIP := flag.String("ip", defaultPrinterIP, "Printer IP Address")
	flag.Parse()

	targetURL := fmt.Sprintf("https://%s/hp/device/this.printservice?printThis", *printerIP)

	// Check Dependencies
	if !checkCommand(gsCommand) {
		fmt.Println("⚠️  WARNING: Ghostscript ('gs') not found. PDF Optimization will fail.")
	}
	if !checkCommand(officeCommand) {
		fmt.Println("⚠️  WARNING: LibreOffice not found. Word/Excel/PPT conversion will fail.")
		fmt.Println("   Install via: sudo apt install libreoffice")
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
	// limit upload size to 50MB
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		render(w, "Error: No file uploaded.", "error")
		return
	}
	defer file.Close()

	// save to temp file
	// Note: LibreOffice is sensitive to file extensions, so keep the original extension (.docx, .xlsx, etc.)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	tempInput, err := os.CreateTemp("", "upload_*"+ext)
	if err != nil {
		render(w, "Server Error: Cannot create temp file.", "error")
		return
	}
	// Note: We don't immediately defer remove here because the path might be needed later; we'll clean up at the end
	defer os.Remove(tempInput.Name())
	defer tempInput.Close()

	_, err = io.Copy(tempInput, file)
	if err != nil {
		render(w, "Server Error: File save failed.", "error")
		return
	}
	tempInput.Close() // Close handle to allow external programs to access

	finalFilePath := tempInput.Name()
	currentExt := ext
	statusMsg := fmt.Sprintf("✅ Success! '%s' sent to printer.", header.Filename)

	// --- Stage 1: If it's an Office file, convert to PDF first ---
	if isOfficeFile(currentExt) {
		fmt.Printf("📄 Detecting Office file (%s). Converting to PDF via LibreOffice...\n", currentExt)

		// Generate PDF path (LibreOffice will generate .pdf in the same directory)
		// This logic is a bit tricky: LibreOffice --outdir specifies the output directory
		outputDir := os.TempDir()

		err := convertOfficeToPDF(tempInput.Name(), outputDir)
		if err != nil {
			fmt.Printf("❌ LibreOffice Failed: %v\n", err)
			render(w, "Error: Office conversion failed. Server might miss fonts or LibreOffice.", "error")
			return
		}

		// Calculate generated PDF filename
		// CreateTemp generates a filename like /tmp/upload_123.docx
		// LibreOffice will generate /tmp/upload_123.pdf
		baseName := strings.TrimSuffix(filepath.Base(tempInput.Name()), currentExt)
		pdfPath := filepath.Join(outputDir, baseName+".pdf")

		// Update current processing file path and extension
		finalFilePath = pdfPath
		currentExt = ".pdf"
		defer os.Remove(pdfPath) // Clean up intermediate file after conversion

		statusMsg += " (Converted to PDF)"
		fmt.Println("✅ Office -> PDF done.")
	}

	// --- Stage 2: If it's a PDF (uploaded or just converted), convert to PS via GS ---
	if currentExt == ".pdf" {
		fmt.Printf("🔄 Optimizing PDF via Ghostscript...\n")

		// Generate PS temp file
		psTemp, _ := os.CreateTemp("", "optimized_*.ps")
		psPath := psTemp.Name()
		psTemp.Close()
		defer os.Remove(psPath)

		err := execGS(finalFilePath, psPath)
		if err != nil {
			fmt.Printf("❌ GS Conversion Failed: %v\n", err)
			render(w, "Error: PDF optimization failed.", "error")
			return
		}

		finalFilePath = psPath
		statusMsg += " (Optimized via Ghostscript)"
		fmt.Println("✅ PDF -> PS done.")
	}

	// --- Stage 3: Send to printer ---
	err = sendToPrinter(finalFilePath, targetURL)
	if err != nil {
		fmt.Printf("❌ Send Failed: %v\n", err)
		render(w, fmt.Sprintf("Printer Error: %v. Check connection.", err), "error")
		return
	}

	render(w, statusMsg, "success")
}

// Check if the file is an Office document
func isOfficeFile(ext string) bool {
	switch ext {
	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return true
	}
	return false
}

// using LibreOffice to convert Office files to PDF
func convertOfficeToPDF(inputPath, outputDir string) error {
	// command: libreoffice --headless --convert-to pdf --outdir /tmp /tmp/file.docx
	cmd := exec.Command(officeCommand,
		"--headless",
		"--convert-to", "pdf",
		"--outdir", outputDir,
		inputPath,
	)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libreoffice error: %v, output: %s", err, string(output))
	}
	return nil
}

func execGS(inputPath, outputPath string) error {
	cmd := exec.Command(gsCommand,
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		"-sDEVICE=ps2write",
		"-dLanguageLevel=2",
		"-sPAPERSIZE=letter",
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
	fmt.Println("⚡ Sending via Curl...")
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

func checkCommand(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}
