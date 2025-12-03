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
	gsCommand     string
	officeCommand string
)

// --- Data Structures for Template ---
type SystemInfo struct {
	OS           string
	Arch         string
	Hostname     string
	GSPath       string
	GSStatus     bool
	OfficePath   string
	OfficeStatus bool
	CurlStatus   bool
}

type PageData struct {
	Message     string
	StatusClass string
	SysInfo     SystemInfo
}

const htmlTemplateStr = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>HP Print Service</title>
    <style>
        :root { --primary: #1a73e8; --bg: #f4f6f8; --text: #202124; --subtext: #5f6368; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--bg); display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; padding: 20px; box-sizing: border-box; }
        .container { width: 100%; max-width: 440px; }
        .card { background: white; padding: 40px; border-radius: 16px; box-shadow: 0 10px 25px rgba(0,0,0,0.05); text-align: center; margin-bottom: 20px; }
        h2 { color: var(--text); margin-bottom: 8px; font-size: 24px; }
        .subtitle { color: var(--subtext); font-size: 14px; margin-bottom: 30px; }

        .upload-area { border: 2px dashed #dadce0; padding: 30px 20px; border-radius: 8px; background: #fff; cursor: pointer; position: relative; transition: all 0.2s; }
        .upload-area:hover { border-color: var(--primary); background: #f8faff; }
        input[type="file"] { position: absolute; top: 0; left: 0; width: 100%; height: 100%; opacity: 0; cursor: pointer; }
        .icon { font-size: 36px; margin-bottom: 10px; display: block; }

        .btn { background: var(--primary); color: white; border: none; padding: 12px 24px; width: 100%; border-radius: 6px; margin-top: 24px; font-size: 16px; font-weight: 500; cursor: pointer; transition: background 0.2s; }
        .btn:hover { background: #1557b0; }
        .btn:disabled { background: #dadce0; cursor: not-allowed; }

        .status { margin-top: 20px; padding: 12px; border-radius: 6px; font-size: 14px; text-align: left; }
        .success { background: #e6f4ea; color: #137333; border: 1px solid #ceead6; }
        .error { background: #fce8e6; color: #c5221f; border: 1px solid #fad2cf; }

        /* System Info Section */
        .sys-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 4px 10px rgba(0,0,0,0.03); font-size: 12px; color: var(--text); text-align: left; }
        .sys-header { font-weight: 700; color: var(--subtext); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 10px; border-bottom: 1px solid #eee; padding-bottom: 8px; }
        .sys-row { display: flex; justify-content: space-between; margin-bottom: 6px; }
        .sys-label { color: var(--subtext); }
        .sys-val { font-family: monospace; font-weight: 600; }
        .badge { padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: bold; }
        .badge-ok { background: #e6f4ea; color: #137333; }
        .badge-err { background: #fce8e6; color: #c5221f; }
        .path-detail { display: block; font-family: monospace; font-size: 10px; color: #999; margin-bottom: 8px; word-break: break-all; }
    </style>
    <script>
        function handleFileSelect(input) {
            const fileName = input.files[0].name;
            document.getElementById('fileLabel').innerText = fileName;
            document.getElementById('submitBtn').disabled = false;
        }
        function showLoading() {
            var btn = document.getElementById('submitBtn');
            btn.disabled = true;
            btn.innerText = "Processing...";
            var status = document.getElementById('statusMsg');
            if(status) status.style.display = 'none';
        }
    </script>
</head>
<body>
    <div class="container">
        <div class="card">
            <h2>🖨️ Print Service</h2>
            <p class="subtitle">Upload File &rarr; Convert &rarr; Print</p>

            <form action="/upload" method="post" enctype="multipart/form-data" onsubmit="showLoading()">
                <div class="upload-area">
                    <span class="icon">📄</span>
                    <p id="fileLabel" style="margin:0; font-weight:500; color:#3c4043;">Select File</p>
                    <p style="font-size: 11px; color: #9aa0a6; margin-top:5px;">PDF, Word, Excel, PowerPoint</p>
                    <input type="file" name="file" required accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx" onchange="handleFileSelect(this)">
                </div>
                <button type="submit" id="submitBtn" class="btn" disabled>Print Now</button>
            </form>

            {{if .Message}}
            <div id="statusMsg" class="status {{.StatusClass}}">
                {{.Message}}
            </div>
            {{end}}
        </div>

        <div class="sys-card">
            <div class="sys-header">System Diagnosis</div>

            <div class="sys-row">
                <span class="sys-label">OS / Arch:</span>
                <span class="sys-val">{{.SysInfo.OS}} / {{.SysInfo.Arch}}</span>
            </div>
            <div class="sys-row">
                <span class="sys-label">Host:</span>
                <span class="sys-val">{{.SysInfo.Hostname}}</span>
            </div>

            <hr style="border: 0; border-top: 1px solid #f0f0f0; margin: 10px 0;">

            <div class="sys-row">
                <span class="sys-label">Ghostscript (PDF opt):</span>
                {{if .SysInfo.GSStatus}}
                    <span class="badge badge-ok">DETECTED</span>
                {{else}}
                    <span class="badge badge-err">MISSING</span>
                {{end}}
            </div>
            <span class="path-detail">{{.SysInfo.GSPath}}</span>

            <div class="sys-row">
                <span class="sys-label">LibreOffice (Doc conv):</span>
                {{if .SysInfo.OfficeStatus}}
                    <span class="badge badge-ok">DETECTED</span>
                {{else}}
                    <span class="badge badge-err">MISSING</span>
                {{end}}
            </div>
            <span class="path-detail">{{.SysInfo.OfficePath}}</span>

             <div class="sys-row">
                <span class="sys-label">Curl (Sender):</span>
                {{if .SysInfo.CurlStatus}}
                    <span class="badge badge-ok">DETECTED</span>
                {{else}}
                    <span class="badge badge-err">MISSING</span>
                {{end}}
            </div>
        </div>
    </div>
</body>
</html>
`

func main() {
	// 1. Set default command paths based on OS
	var defaultGSPath, defaultOfficePath string

	switch runtime.GOOS {
	case "windows":
		// Windows default paths
		defaultGSPath = "gswin64c"
		defaultOfficePath = `C:\Program Files\LibreOffice\program\soffice.exe`
	case "darwin":
		defaultGSPath = "gs"
		defaultOfficePath = "/Applications/LibreOffice.app/Contents/MacOS/soffice"
	default: // Linux and others
		defaultGSPath = "gs"
		defaultOfficePath = "libreoffice"
	}

	// 2. Define command-line flags
	port := flag.String("port", "8080", "Server Port")
	printerIP := flag.String("ip", "10.31.6.225", "Printer IP Address")

	// New flags: allow overriding default paths
	flagGS := flag.String("gs", defaultGSPath, "Path to Ghostscript executable")
	flagOffice := flag.String("office", defaultOfficePath, "Path to LibreOffice/OpenOffice executable")

	flag.Parse()

	// 3. Assign parsed flags to global variables
	gsCommand = *flagGS
	officeCommand = *flagOffice

	targetURL := fmt.Sprintf("https://%s/hp/device/this.printservice?printThis", *printerIP)

	printConsoleStartupInfo(*port, *printerIP)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			render(w, "", "")
			return
		}
		handleUpload(w, r, targetURL)
	})

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

	ext := strings.ToLower(filepath.Ext(header.Filename))
	tempInput, err := os.CreateTemp("", "upload_*"+ext)
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
	currentExt := ext
	statusMsg := fmt.Sprintf("✅ Success! '%s' sent to printer.", header.Filename)

	// --- Stage 1: Office -> PDF ---
	if isOfficeFile(currentExt) {
		fmt.Printf("📄 Processing Office file: %s\n", header.Filename)
		outputDir := os.TempDir()

		err := convertOfficeToPDF(tempInput.Name(), outputDir)
		if err != nil {
			fmt.Printf("❌ LibreOffice Failed: %v\n", err)
			render(w, "Error: Office conversion failed. Check server logs.", "error")
			return
		}

		baseName := strings.TrimSuffix(filepath.Base(tempInput.Name()), currentExt)
		pdfPath := filepath.Join(outputDir, baseName+".pdf")

		finalFilePath = pdfPath
		currentExt = ".pdf"
		defer os.Remove(pdfPath)
		statusMsg += " (Converted to PDF)"
	}

	// --- Stage 2: PDF -> PS (GS) ---
	if currentExt == ".pdf" {
		fmt.Printf("🔄 Optimizing PDF...\n")
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
		statusMsg += " (Optimized)"
	}

	// --- Stage 3: Print ---
	err = sendToPrinter(finalFilePath, targetURL)
	if err != nil {
		fmt.Printf("❌ Send Failed: %v\n", err)
		render(w, fmt.Sprintf("Printer Error: %v", err), "error")
		return
	}

	render(w, statusMsg, "success")
}

func render(w http.ResponseWriter, msg, statusClass string) {
	sysInfo := getSystemInfo()

	data := PageData{
		Message:     msg,
		StatusClass: statusClass,
		SysInfo:     sysInfo,
	}

	t, err := template.New("page").Parse(htmlTemplateStr)
	if err != nil {
		http.Error(w, "Template Error", 500)
		return
	}
	t.Execute(w, data)
}

// Fetch system information
func getSystemInfo() SystemInfo {
	host, _ := os.Hostname()
	return SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Hostname:     host,
		GSPath:       gsCommand,
		GSStatus:     checkCommand(gsCommand),
		OfficePath:   officeCommand,
		OfficeStatus: checkCommand(officeCommand),
		CurlStatus:   checkCommand("curl"),
	}
}

func printConsoleStartupInfo(port, printerIP string) {
	info := getSystemInfo()
	fmt.Println("\n========================================")
	fmt.Printf("🚀 Print Server Started on :%s\n", port)
	fmt.Printf("🖨️  Target Printer IP: %s\n", printerIP)
	fmt.Println("----------------------------------------")
	fmt.Printf("🖥️  System: %s/%s (%s)\n", info.OS, info.Arch, info.Hostname)

	fmt.Printf("📄 Ghostscript:  ")
	if info.GSStatus {
		fmt.Println("✅ Found")
	} else {
		fmt.Println("❌ Not Found")
	}

	fmt.Printf("📊 LibreOffice:  ")
	if info.OfficeStatus {
		fmt.Println("✅ Found")
	} else {
		fmt.Println("❌ Not Found")
	}

	fmt.Printf("🌐 Curl Utility: ")
	if info.CurlStatus {
		fmt.Println("✅ Found")
	} else {
		fmt.Println("❌ Not Found (Required)")
	}
	fmt.Println("========================================")
}

func isOfficeFile(ext string) bool {
	switch ext {
	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return true
	}
	return false
}

func convertOfficeToPDF(inputPath, outputDir string) error {
	// Use global variable officeCommand
	cmd := exec.Command(officeCommand,
		"--headless",
		"--convert-to", "pdf",
		"--outdir", outputDir,
		inputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("err: %v, out: %s", err, string(output))
	}
	return nil
}

func execGS(inputPath, outputPath string) error {
	// Use global variable gsCommand
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
		return fmt.Errorf("err: %v, log: %s", err, stderr.String())
	}
	return nil
}

func sendToPrinter(filePath, url string) error {
	fmt.Println("⚡ Sending via Curl...")
	// Note: On Windows, curl must be available (Win10+ includes it)
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

// Test if a command exists in PATH or at given path
func checkCommand(cmdPath string) bool {
	// 1. If it contains a path separator, check if the file exists directly
	if strings.Contains(cmdPath, string(os.PathSeparator)) {
		_, err := os.Stat(cmdPath)
		return err == nil
	}
	// 2. Otherwise, look in PATH
	_, err := exec.LookPath(cmdPath)
	return err == nil
}
