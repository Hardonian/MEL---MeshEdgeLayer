package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mel-project/mel/internal/config"
	"github.com/mel-project/mel/internal/db"
	"github.com/mel-project/mel/internal/policy"
	"github.com/mel-project/mel/internal/privacy"
	"github.com/mel-project/mel/internal/service"
	"github.com/mel-project/mel/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "version":
		fmt.Println(version.Version)
	case "config":
		configCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	case "doctor":
		doctorCmd(os.Args[2:])
	case "status":
		apiGet(os.Args[2:], "/api/status")
	case "nodes":
		apiGet(os.Args[2:], "/api/nodes")
	case "transports":
		transportsCmd(os.Args[2:])
	case "db":
		dbCmd(os.Args[2:])
	case "export":
		exportCmd(os.Args[2:])
	case "logs":
		logsCmd(os.Args[2:])
	case "policy":
		policyCmd(os.Args[2:])
	case "privacy":
		privacyCmd(os.Args[2:])
	case "dev-simulate-mqtt":
		simulateCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}
func usage() {
	fmt.Println(`mel commands: version, doctor, config validate, serve, status, nodes, transports list, db vacuum, export, logs tail, policy explain, privacy audit, dev-simulate-mqtt`)
}
func fs(name string) *flag.FlagSet { f := flag.NewFlagSet(name, flag.ExitOnError); return f }
func loadCfg(args []string) (config.Config, *flag.FlagSet) {
	f := fs("cfg")
	path := f.String("config", "configs/mel.example.json", "config")
	_ = f.Parse(args)
	cfg, _, err := config.Load(*path)
	if err != nil {
		panic(err)
	}
	return cfg, f
}
func configCmd(args []string) {
	if len(args) == 0 || args[0] != "validate" {
		panic("usage: mel config validate --config <path>")
	}
	cfg, _ := loadCfg(args[1:])
	if err := config.Validate(cfg); err != nil {
		panic(err)
	}
	fmt.Println("config valid")
}
func serveCmd(args []string) {
	cfg, _ := loadCfg(args)
	app, err := service.New(cfg)
	if err != nil {
		panic(err)
	}
	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
}
func doctorCmd(args []string) {
	cfg, _ := loadCfg(args)
	findings := []string{}
	if err := config.Validate(cfg); err != nil {
		findings = append(findings, "config: "+err.Error())
	}
	if _, err := db.Open(cfg); err != nil {
		findings = append(findings, "db: "+err.Error())
	}
	if _, err := os.Stat(cfg.Storage.DataDir); err != nil {
		findings = append(findings, "data_dir: "+err.Error())
	}
	if cfg.Bind.AllowRemote && !cfg.Auth.Enabled {
		findings = append(findings, "remote bind without auth")
	}
	if len(findings) == 0 {
		fmt.Println("doctor ok")
		return
	}
	b, _ := json.MarshalIndent(findings, "", "  ")
	fmt.Println(string(b))
	os.Exit(1)
}
func apiGet(args []string, path string) {
	f := fs("api")
	addr := f.String("addr", "http://127.0.0.1:8080", "addr")
	_ = f.Parse(args)
	resp, err := http.Get(*addr + path)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var body any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	out, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println(string(out))
}
func transportsCmd(args []string) {
	if len(args) == 0 || args[0] != "list" {
		panic("usage: mel transports list")
	}
	cfg, _ := loadCfg(args[1:])
	out, _ := json.MarshalIndent(cfg.Transports, "", "  ")
	fmt.Println(string(out))
}
func dbCmd(args []string) {
	if len(args) == 0 || args[0] != "vacuum" {
		panic("usage: mel db vacuum")
	}
	cfg, _ := loadCfg(args[1:])
	d, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	if err := d.Vacuum(); err != nil {
		panic(err)
	}
	fmt.Println("vacuum complete")
}
func exportCmd(args []string) {
	cfg, _ := loadCfg(args)
	d, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	rows, err := d.QueryJSON("SELECT node_num,node_id,long_name,short_name,last_seen,lat_redacted,lon_redacted,altitude FROM nodes ORDER BY node_num;")
	if err != nil {
		panic(err)
	}
	out := map[string]any{"nodes": rows}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
func logsCmd(args []string) {
	if len(args) == 0 || args[0] != "tail" {
		panic("usage: mel logs tail")
	}
	cfg, _ := loadCfg(args[1:])
	d, err := db.Open(cfg)
	if err != nil {
		panic(err)
	}
	rows, err := d.QueryJSON("SELECT category,level,message,created_at FROM audit_logs ORDER BY id DESC LIMIT 20;")
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(rows, "", "  ")
	fmt.Println(string(b))
}
func policyCmd(args []string) {
	if len(args) == 0 || args[0] != "explain" {
		panic("usage: mel policy explain")
	}
	cfg, _ := loadCfg(args[1:])
	b, _ := json.MarshalIndent(policy.Explain(cfg), "", "  ")
	fmt.Println(string(b))
}
func privacyCmd(args []string) {
	if len(args) == 0 || args[0] != "audit" {
		panic("usage: mel privacy audit")
	}
	cfg, _ := loadCfg(args[1:])
	b, _ := json.MarshalIndent(privacy.Audit(cfg), "", "  ")
	fmt.Println(string(b))
}
func simulateCmd(args []string) {
	f := fs("sim")
	endpoint := f.String("endpoint", "127.0.0.1:18830", "endpoint")
	topic := f.String("topic", "msh/US/2/e/test", "topic")
	_ = f.Parse(args)
	env := sampleEnvelope()
	runMQTTServer(*endpoint, *topic, env)
}
func sampleEnvelope() []byte {
	user := msg(
		fieldBytes(1, []byte("!abcd1234")),
		fieldBytes(2, []byte("Relay Node")),
		fieldBytes(3, []byte("RN")),
	)
	data := msg(fieldVarint(1, 4), fieldBytes(2, user))
	packet := msg(fieldFixed32(1, 12345), fieldFixed32(2, 255), fieldMsg(4, data), fieldFixed32(6, 99), fieldFixed32(7, uint32(time.Now().Unix())), fieldVarint(9, 3), fieldVarint(12, 42))
	env := msg(fieldMsg(1, packet), fieldBytes(2, []byte("mel-test")), fieldBytes(3, []byte("!gateway")))
	return env
}
func msg(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
func tag(field int, wt int) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, uint64(field<<3|wt))
	return b[:n]
}
func fieldVarint(field int, v uint64) []byte {
	b := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(b, v)
	return append(tag(field, 0), b[:n]...)
}
func fieldFixed32(field int, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(tag(field, 5), b...)
}
func fieldBytes(field int, v []byte) []byte {
	ln := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(ln, uint64(len(v)))
	out := append(tag(field, 2), ln[:n]...)
	return append(out, v...)
}
func fieldMsg(field int, v []byte) []byte { return fieldBytes(field, v) }
func runMQTTServer(endpoint, topic string, payload []byte) {
	ln, err := netListen(endpoint)
	if err != nil {
		panic(err)
	}
	defer ln.Close()
	conn, err := ln.Accept()
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	hdr := make([]byte, 2)
	_, _ = conn.Read(hdr)
	rem := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, _ := conn.Read(rem)
	_ = n
	conn.Write([]byte{0x20, 0x02, 0x00, 0x00})
	_, _ = conn.Read(rem)
	publish := bytes.NewBuffer([]byte{0x30})
	body := bytes.NewBuffer(nil)
	binary.Write(body, binary.BigEndian, uint16(len(topic)))
	body.WriteString(topic)
	body.Write(payload)
	remaining(body.Len(), publish)
	publish.Write(body.Bytes())
	conn.Write(publish.Bytes())
	time.Sleep(2 * time.Second)
}
func remaining(n int, buf *bytes.Buffer) {
	for {
		d := byte(n % 128)
		n /= 128
		if n > 0 {
			d |= 128
		}
		buf.WriteByte(d)
		if n == 0 {
			break
		}
	}
}
func netListen(addr string) (interface {
	Accept() (net.Conn, error)
	Close() error
}, error) {
	return net.Listen("tcp", addr)
}

var _ = exec.Command
var _ = strings.Builder{}
var _ = base64.StdEncoding
