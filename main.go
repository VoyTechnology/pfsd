package main

import (
	"flag"
	"fmt"
	"github.com/cpssd/paranoid/logger"
	"github.com/cpssd/paranoid/pfsd/dnetclient"
	"github.com/cpssd/paranoid/pfsd/globals"
	"github.com/cpssd/paranoid/pfsd/pfi"
	"github.com/cpssd/paranoid/pfsd/pnetclient"
	"github.com/cpssd/paranoid/pfsd/pnetserver"
	"github.com/cpssd/paranoid/pfsd/upnp"
	pb "github.com/cpssd/paranoid/proto/paranoidnetwork"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	srv *grpc.Server
	log *logger.ParanoidLogger

	certFile   = flag.String("cert", "", "TLS certificate file - if empty connection will be unencrypted")
	keyFile    = flag.String("key", "", "TLS key file - if empty connection will be unencrypted")
	noNetwork  = flag.Bool("no_networking", false, "Do not perform any networking")
	skipVerify = flag.Bool("skip_verification", false,
		"skip verification of TLS certificate chain and hostname - not recommended unless using self-signed certs")
	verbose = flag.Bool("v", false, "Use verbose logging")
)

func startRPCServer(lis *net.Listener) {
	var opts []grpc.ServerOption
	if globals.TLSEnabled {
		log.Info("Starting ParanoidNetwork server with TLS.")
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatal("Failed to generate TLS credentials:", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	} else {
		log.Info("Starting ParanoidNetwork server without TLS.")
	}
	srv = grpc.NewServer(opts...)
	pb.RegisterParanoidNetworkServer(srv, &pnetserver.ParanoidServer{})
	globals.Wait.Add(1)
	go srv.Serve(*lis)
}

func main() {
	flag.Parse()

	if len(flag.Args()) < 4 {
		fmt.Print("Usage:\n\tpfsd <paranoid_directory> <mount_point> <Discovery Server> <Discovery Port>\n")
		os.Exit(1)
	}
	log = logger.New("main", "pfsd", path.Join(flag.Arg(0), "meta", "logs"))
	dnetclient.Log = logger.New("dnetclient", "pfsd", path.Join(flag.Arg(0), "meta", "logs"))
	pnetclient.Log = logger.New("pnetclient", "pfsd", path.Join(flag.Arg(0), "meta", "logs"))
	pnetserver.Log = logger.New("pnetserver", "pfsd", path.Join(flag.Arg(0), "meta", "logs"))
	upnp.Log = logger.New("upnp", "pfsd", path.Join(flag.Arg(0), "meta", "logs"))

	log.SetOutput(logger.STDERR | logger.LOGFILE)
	dnetclient.Log.SetOutput(logger.STDERR | logger.LOGFILE)
	pnetclient.Log.SetOutput(logger.STDERR | logger.LOGFILE)
	pnetserver.Log.SetOutput(logger.STDERR | logger.LOGFILE)
	upnp.Log.SetOutput(logger.STDERR | logger.LOGFILE)

	globals.TLSSkipVerify = *skipVerify
	if *certFile != "" && *keyFile != "" {
		globals.TLSEnabled = true
		if !globals.TLSSkipVerify {
			cn, err := getCommonNameFromCert(*certFile)
			if err != nil {
				log.Fatal("Could not get CN from provided TLS cert:", err)
			}
			globals.CommonName = cn
		}
	} else {
		globals.TLSEnabled = false
	}

	if !*noNetwork {
		discoveryPort, err := strconv.Atoi(flag.Arg(3))
		if err != nil || discoveryPort < 1 || discoveryPort > 65535 {
			log.Fatal("Discovery port must be a number between 1 and 65535, inclusive.")
		}
	}

	if !*noNetwork {
		pnetserver.ParanoidDir = flag.Arg(0)

		ip, err := upnp.GetIP()
		if err != nil {
			log.Fatalln("FATAL: Could not get IP:", err)
		}
		//Asking for port 0 requests a random free port from the OS.
		lis, err := net.Listen("tcp", ip+":0")
		if err != nil {
			log.Fatalf("Failed to start listening : %v.\n", err)
		}
		splits := strings.Split(lis.Addr().String(), ":")
		port, err := strconv.Atoi(splits[len(splits)-1])
		if err != nil {
			log.Fatal("Could not parse port", splits[len(splits)-1], " Error :", err)
		}
		globals.Port = port

		//Try and set up uPnP. Otherwise use internal IP.
		globals.UPnPEnabled = false
		err = upnp.DiscoverDevices()
		if err == nil {
			log.Info("UPnP devices available")
			externalPort, err := upnp.AddPortMapping(port)
			if err == nil {
				log.Info("UPnP port mapping enabled")
				port = externalPort
				globals.Port = externalPort
				globals.UPnPEnabled = true
			}
		}

		globals.Server, err = upnp.GetIP()
		if err != nil {
			log.Fatal("Can't get IP. Error : ", err)
		}
		log.Info("Peer address:", globals.Server+":"+strconv.Itoa(globals.Port))

		if _, err := os.Stat(pnetserver.ParanoidDir); os.IsNotExist(err) {
			log.Fatal("Path", pnetserver.ParanoidDir, "does not exist.")
		}
		if _, err := os.Stat(path.Join(pnetserver.ParanoidDir, "meta")); os.IsNotExist(err) {
			log.Fatal("Path", pnetserver.ParanoidDir, "is not valid PFS root.")
		}

		dnetclient.SetDiscovery(flag.Arg(2), flag.Arg(3), strconv.Itoa(port))
		dnetclient.JoinDiscovery("_")
		startRPCServer(&lis)
	}
	createPid("pfsd")
	globals.Wait.Add(1)
	go pfi.StartPfi(flag.Arg(0), flag.Arg(1), *verbose, !*noNetwork)
	HandleSignals()
}

func createPid(processName string) {
	processID := os.Getpid()
	pid := []byte(strconv.Itoa(processID))
	err := ioutil.WriteFile(path.Join(pnetserver.ParanoidDir, "meta", processName+".pid"), pid, 0600)
	if err != nil {
		log.Fatal("Failed to create PID file", err)
	}
}
