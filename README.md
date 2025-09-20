# Simple Upload Server

A lightweight, high-performance file upload server built in Go that supports resumable uploads for large files. Designed for simplicity and reliability, it provides both a web interface and API endpoints for uploading files of any size.

## Features

### 🚀 **High Performance**
- **HTTP/3 Support**: Automatic HTTP/3 (QUIC) when TLS is enabled for maximum performance
- **HTTP/2 & HTTP/1.1 Fallback**: Seamless compatibility with older clients
- **Resumable Uploads**: Built on the [TUS protocol](https://tus.io/) - never lose progress on large uploads

### 📁 **File Management**
- **Automatic File Renaming**: Files are renamed from internal IDs to original filenames upon completion
- **Filename Sanitization**: Unsafe characters are automatically cleaned for filesystem safety
- **Duplicate Handling**: Automatic filename conflict resolution with numbered suffixes
- **Large File Support**: No artificial file size limits - upload files of any size

### 🔒 **Security & Reliability**
- **TLS/HTTPS Support**: Full SSL/TLS encryption with automatic HTTP/3 upgrade
- **Alt-Svc Headers**: Automatic HTTP/3 advertisement for compatible clients  
- **Safe File Handling**: Comprehensive filename sanitization and validation
- **Detailed Logging**: Complete upload tracking and error reporting

### 🌐 **Web Interface**
- **Modern UI**: Clean, responsive web interface for easy file uploads
- **Progress Tracking**: Real-time upload progress with resumable capability
- **Drag & Drop**: Intuitive file selection and upload experience

## Quick Start

### Basic Usage

#### HTTP Server
```bash
./simple-upload --port 8080 --uploads-dir ./uploads
```
Open http://localhost:8080 in your browser

#### HTTPS Server with HTTP/3 
```bash
./simple-upload --port 8443 --cert server.crt --key server.key --uploads-dir ./uploads
```
Open https://localhost:8443 in your browser

## Command Line Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `8080` | Port to listen on |
| `--uploads-dir` | `-d` | `./uploads` | Directory to store uploaded files |
| `--cert` | `-c` | | Path to TLS certificate file (enables HTTPS and HTTP/3) |
| `--key` | `-k` | | Path to TLS private key file (enables HTTPS and HTTP/3) |
| `--help` | `-h` | | Show help information |

## API Usage

The server implements the [TUS resumable upload protocol](https://tus.io/protocols/resumable-upload.html).

### Upload Endpoints
- `POST /files/` - Create new upload
- `PATCH /files/{id}` - Resume upload
- `HEAD /files/{id}` - Check upload status
- `GET /` - Web interface

### Example with curl
```bash
# Create upload
curl -X POST http://localhost:8080/files/ \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Length: 1000000" \
  -H "Upload-Metadata: filename $(echo -n 'large-file.zip' | base64)"

# Upload data
curl -X PATCH http://localhost:8080/files/{upload-id} \
  -H "Tus-Resumable: 1.0.0" \
  -H "Upload-Offset: 0" \
  -H "Content-Type: application/offset+octet-stream" \
  --data-binary @large-file.zip
```

## Advanced Configuration


### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        
        # Important for TUS protocol
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Disable request buffering for large uploads
        proxy_request_buffering off;
        proxy_buffering off;
        
        # Increase timeouts for large uploads
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

## How It Works

### Upload Process
1. **Initiate**: Client creates upload with `POST /files/` including file metadata
2. **Upload**: File data sent in chunks via `PATCH /files/{id}` requests
3. **Resume**: If interrupted, client can resume from last uploaded byte
4. **Complete**: Server automatically renames file from ID to original filename
5. **Cleanup**: Temporary upload state is cleaned up

### File Naming
- **During Upload**: Files stored with unique upload ID
- **After Completion**: Automatically renamed to original filename
- **Conflict Resolution**: Duplicate names get numbered suffix (`file_1.txt`, `file_2.txt`)
- **Safety**: Unsafe characters (`/`, `\`, `..`, etc.) are sanitized

### Protocol Support
- **HTTP/3**: Automatically enabled with TLS certificates
- **HTTP/2**: Available with TLS certificates  
- **HTTP/1.1**: Always available as fallback
- **Alt-Svc**: Headers automatically advertise HTTP/3 to compatible clients

## Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# Check what's using the port
sudo lsof -i :8080

# Use a different port
./simple-upload --port 8081
```

#### Permission Denied (Uploads Directory)
```bash
# Create directory with proper permissions
mkdir -p ./uploads
chmod 755 ./uploads

# Or specify a different directory
./simple-upload --uploads-dir /tmp/uploads
```

#### TLS Certificate Issues
```bash
# Verify certificate
openssl x509 -in server.crt -text -noout

# Check certificate and key match
openssl x509 -noout -modulus -in server.crt | openssl md5
openssl rsa -noout -modulus -in server.key | openssl md5
```

### Logs and Debugging

The server provides detailed structured logging:
```bash
# Run with debug output
./simple-upload --port 8080 2>&1 | jq
```

Log entries include:
- Upload start/completion events
- File renaming operations
- HTTP/3 connection attempts
- Error details with context

## Development

### Building the UI
```bash
cd ui
npm install
npm run build
```

### Project Structure
```
simple-upload/
├── main.go              # Main server application
├── go.mod              # Go dependencies
├── ui/                 # Web interface source
│   ├── src/           # Frontend source code
│   ├── dist/          # Built frontend assets
│   └── package.json   # Node.js dependencies
├── uploads/           # Default upload directory
└── README.md          # This file
```

### Dependencies
- **[quic-go](https://github.com/quic-go/quic-go)**: HTTP/3 support
- **[tusd](https://github.com/tus/tusd)**: TUS resumable upload protocol
- **[cobra](https://github.com/spf13/cobra)**: CLI interface

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Support

- 📚 [TUS Protocol Documentation](https://tus.io/protocols/resumable-upload.html)
- 🐛 [Report Issues](https://github.com/yourusername/simple-upload/issues)
- 💡 [Feature Requests](https://github.com/yourusername/simple-upload/discussions)

---

**Simple Upload Server** - Built for reliability, designed for simplicity.