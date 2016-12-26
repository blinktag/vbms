package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"database/sql"

	"github.com/Sirupsen/logrus"
	fastping "github.com/tatsushid/go-fastping"
)

// Server holds details for current server
type Server struct {
	ID          int    `sql:"id"`
	Hostname    string `sql:"hostname"`
	IP          string `sql:"ip"`
	EnableHTTP  bool   `sql:"enablehttp"`
	ResultHTTP  string `sql:"httpresult"`
	EnableSMTP  bool   `sql:"enablestmp"`
	ResultSMTP  string `sql:"smtpresult"`
	PortSMTP    int    `sql:"smtpport"`
	EnablePOP3  bool   `sql:"enablepop3"`
	ResultPOP3  string `sql:"pop3result"`
	EnableHTTPS bool   `sql:"enablehttps"`
	ResultHTTPS string `sql:"httpsresult"`
	EnablePing  bool   `sql:"enableping"`
	ResultPing  string `sql:"pingresult"`
	DB          *sql.DB
}

// GetLogger returns instance of logrus prepopulated with server fields
func (s *Server) GetLogger(service string, port int) *logrus.Entry {
	contextLogger := logrus.WithFields(logrus.Fields{
		"Server":  s.Hostname,
		"Service": service,
		"Port":    port,
	})

	return contextLogger
}

// CheckHTTP opens connection on port 80 and checks for HTTP response
func (s *Server) CheckHTTP(wg *sync.WaitGroup) {

	defer wg.Done()

	if !s.EnableHTTP {
		return
	}

	logger := s.GetLogger("HTTP", 80)

	// Open connection on port 80
	conn, err := net.Dial("tcp", s.IP+":80")
	if err != nil {
		s.ResultHTTP = "Unable to open port"
		logger.WithError(err).Error(s.ResultHTTP)
		return
	}

	// Ensure we close after returning
	defer conn.Close()

	// Send basic GET request
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

	// Read first line response
	result, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.ResultHTTP = "No response received from server"
		logger.Error(s.ResultHTTP)
		return
	}

	// Expect response of "HTTP/1.1 200 OK"
	result = strings.TrimSpace(result)
	s.ResultHTTP = result

	if isValidHTTPResponse(result) {
		logger.Infof("HTTP Check Ok. Response: %v", result)
	} else {
		logger.Errorf("Returned invalid HTTPS response: '%v'", result)
	}
}

// CheckHTTPS opens connection on port 80 and checks for HTTP response
func (s *Server) CheckHTTPS(wg *sync.WaitGroup) {

	defer wg.Done()

	if !s.EnableHTTPS {
		return
	}

	logger := s.GetLogger("HTTPS", 443)

	dialer := &net.Dialer{Timeout: time.Second * 3}

	// Open connection on port 443
	conn, err := tls.DialWithDialer(dialer, "tcp", s.Hostname+":443", &tls.Config{})
	if err != nil {
		s.ResultHTTPS = "Unable to open port"
		logger.WithError(err).Error(s.ResultHTTPS)
		return
	}

	// Ensure we close after returning
	defer conn.Close()

	// Send basic GET request
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

	// Read first line response
	result, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.ResultHTTPS = "No response received from server"
		logger.Error(s.ResultHTTPS)
		return
	}

	// Expect response of "HTTP/1.1 200 OK"
	result = strings.TrimSpace(result)
	s.ResultHTTPS = result

	if isValidHTTPResponse(result) {
		logger.Infof("HTTP Check Ok. Response: %v", result)
	} else {
		logger.Errorf("Returned invalid HTTPS response: '%v'", result)
	}
}

// isValidHTTPResponse checks if HTTP resonse from server is HTTP code 200
func isValidHTTPResponse(response string) bool {
	re := regexp.MustCompile("200 OK")
	return re.FindString(response) != ""
}

// CheckSMTP sends HELO to STMP server and expects a response
func (s *Server) CheckSMTP(wg *sync.WaitGroup) {

	defer wg.Done()

	if !s.EnableSMTP {
		return
	}

	logger := s.GetLogger("SMTP", s.PortSMTP)

	// Convert port to string for concatenation
	port := strconv.Itoa(s.PortSMTP)

	// Open connection
	conn, err := net.DialTimeout("tcp", s.IP+port, 10*time.Second)

	// Log failure
	if err != nil {
		s.ResultSMTP = "Unable to open SMTP connection"
		logger.Error(s.ResultSMTP)
		return
	}

	// Make sure we close connection after function returns
	defer conn.Close()

	// Read first line
	result, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.ResultSMTP = "No response received from server"
		logger.Error(s.ResultSMTP)
		return
	}
	result = strings.TrimSpace(result)

	s.ResultSMTP = result

	logger.Infof("SMTP Check OK. Response: %v", result)
}

// CheckPOP3 opens connection on port 80 and checks for HTTP response
func (s *Server) CheckPOP3(wg *sync.WaitGroup) {

	defer wg.Done()

	if !s.EnablePOP3 {
		return
	}

	logger := s.GetLogger("POP3", 110)

	// Open connection on port 80
	conn, err := net.Dial("tcp", s.IP+":110")
	if err != nil {
		s.ResultPOP3 = "Unable to open POP3 Connection"
		logger.Error(s.ResultPOP3)
		return
	}

	// Ensure we close after returning
	defer conn.Close()

	// Send basic GET request
	fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")

	// Read first line of response
	result, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.ResultPOP3 = "No response received from server"
		logger.Error(s.ResultPOP3)
		return
	}

	result = strings.TrimSpace(result)

	s.ResultPOP3 = result

	logger.Infof("Returned on port 110: %v", result)
}

// CheckPing pings the server and expects a response.
func (s *Server) CheckPing(wg *sync.WaitGroup) {

	defer wg.Done()

	if !s.EnablePing {
		return
	}

	logger := s.GetLogger("PING", 0)

	// Check if we're UID of 0
	if os.Getuid() != 0 {
		s.ResultPing = "Ping requires root"
		logger.Error(s.ResultPing)
		return
	}

	// We haven't received ping yet
	received := false

	p := fastping.NewPinger()
	ra, _ := net.ResolveIPAddr("ip4:icmp", s.IP)
	p.AddIPAddr(ra)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		received = true
		s.ResultPing = fmt.Sprintf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
	}

	err := p.Run()
	if err != nil {
		fmt.Println(err)
	}

	if received {
		logger.Info("Ping successful")
	} else {
		logger.Error("Ping failed")
	}
}

// UpdateDatabase commits current state of the server struct to the database
func (s *Server) UpdateDatabase() {

	db := s.DB

	stmt, err := db.Prepare(`
				UPDATE servers
				SET httpresult = ?,
					smtpresult = ?,
					pop3result = ?,
					httpsresult = ?,
					pingresult = ?
				WHERE id = ?
			`)

	if err != nil {
		log.Panic(err)
	}

	_, err = stmt.Exec(s.ResultHTTP, s.ResultSMTP, s.ResultPOP3, s.ResultHTTPS, s.ResultPing, s.ID)

	if err != nil {
		log.Panic(err)
	}
}

// RunChecks initiates all service checks for a server in goroutines
func (s *Server) RunChecks() {

	wg := new(sync.WaitGroup)

	wg.Add(5)
	go s.CheckHTTP(wg)
	go s.CheckSMTP(wg)
	go s.CheckPOP3(wg)
	go s.CheckHTTPS(wg)
	go s.CheckPing(wg)
	s.UpdateDatabase()
	wg.Wait()
}
