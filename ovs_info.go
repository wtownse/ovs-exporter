package main

import (
    "os/exec"
    "fmt"
    "os"
    "io"
    "strconv"
    "encoding/xml"
    "encoding/json"
    "strings"
    "encoding/csv"
    "gopkg.in/alecthomas/kingpin.v2"
    "github.com/digitalocean/go-openvswitch/ovs"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
    log "github.com/Sirupsen/logrus"
)

type Data struct {
	Interfaces	[]Interfaces	`xml:"interface"`
}

type Interfaces	struct { 
	Name	string	`xml:"name,attr"`
	Chassis	Chassis	`xml:"chassis"`
	Port	Port	`xml:"port"`
}

type Chassis struct {
	Id	string	`xml:"id"`
	Name	string	`xml:"name"`
	Desc	string	`xml:"descr"`
	MgmtIp	string	`xml:"mgmt-ip"`
}

type Port struct {
	Id	string	`xml:"id"`
	Desc	string	`xml:"descr"`
	Ttl	string	`xml:"ttl"`
}

type ovsCollector struct {
	bondInfo			*prometheus.Desc
	ovsIntInfo			*prometheus.Desc
	LLDPInterfaceInfo		*prometheus.Desc
	ovsIntStatTxBytesDesc		*prometheus.Desc
	ovsIntStatRxBytesDesc		*prometheus.Desc
	ovsIntStatTxPacketsDesc		*prometheus.Desc
	ovsIntStatRxPacketsDesc		*prometheus.Desc
	ovsIntStatTxDroppedDesc		*prometheus.Desc
	ovsIntStatRxDroppedDesc		*prometheus.Desc
	ovsIntStatTxErrorsDesc		*prometheus.Desc
	ovsIntStatRxErrorsDesc		*prometheus.Desc
	ovsIntStatRxCrcErrDesc		*prometheus.Desc
	ovsIntStatRxFrameErrDesc	*prometheus.Desc
	ovsIntStatRxOverErrDesc		*prometheus.Desc
	ovsIntHwDesc			*prometheus.Desc

}

func newovsCollector() *ovsCollector  {
        return &ovsCollector   {
                bondInfo: prometheus.NewDesc("bondInfo",
                        "Shows ovs bond information",
                        []string{"local_host","bond","slaves","type"},
                        nil,
                ),
                ovsIntInfo: prometheus.NewDesc("ovsIntInfo",
                        "Shows ovs interface information",
                        []string{"local_host","uuid","name","admin_state","link_state","link_speed",
                        "error","duplex","link_resets","mtu","mac_in_use","ofport","status",
                        "type"},
                        nil,
                ),
		LLDPInterfaceInfo: prometheus.NewDesc("LLDPInterfaceInfo",
			"Shows lldp neighbor information",
			[]string{"local_host","local_iface","chassis_mac","chassis_name","chassis_desc","chassis_mgmt_ip","remote_iface","port_desc"},
			nil,
		),
                ovsIntStatTxBytesDesc: prometheus.NewDesc("ovsIntStatTxBytes",
                        "Shows ovs tx_bytes",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxBytesDesc: prometheus.NewDesc("ovsIntStatRxBytes",
                        "Shows ovs rx_bytes",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatTxPacketsDesc: prometheus.NewDesc("ovsIntStatTxPackets",
                        "Shows ovs tx_packets",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxPacketsDesc: prometheus.NewDesc("ovsIntStatRxPackets",
                        "Shows ovs rx_packets",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatTxDroppedDesc: prometheus.NewDesc("ovsIntStatTxDropped",
                        "Shows ovs tx_packets",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxDroppedDesc: prometheus.NewDesc("ovsIntStatRxDropped",
                        "Shows ovs rx_packets",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatTxErrorsDesc: prometheus.NewDesc("ovsIntStatTxErrors",
                        "Shows ovs tx_errors",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxErrorsDesc: prometheus.NewDesc("ovsIntStatRxErrors",
                        "Shows ovs rx_errors",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxCrcErrDesc: prometheus.NewDesc("ovsIntStatRxCrcErr",
                        "Shows ovs rx crc errors",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxFrameErrDesc: prometheus.NewDesc("ovsIntStatRxFrameErr",
                        "Shows ovs rx frame errors",
                        []string{"name","local_host","port"},
                        nil,
                ),
                ovsIntStatRxOverErrDesc: prometheus.NewDesc("ovsIntStatRxOverErr",
                        "Shows ovs rx over runs",
                        []string{"name","local_host","port"},
                        nil,
                ),
		ovsIntHwDesc:  prometheus.NewDesc("ovsIntHw",
			"Shows driver and firmware info",
			[]string{"local_host","port","driver_name","driver_version","firmware_version"},
			nil,
		),
        }
}

func (collector *ovsCollector) Describe(ch chan<- *prometheus.Desc) {
        ch <- collector.bondInfo
        ch <- collector.ovsIntInfo
	ch <- collector.ovsIntHwDesc
	ch <- collector.LLDPInterfaceInfo
        ch <- collector.ovsIntStatTxBytesDesc
        ch <- collector.ovsIntStatRxBytesDesc
        ch <- collector.ovsIntStatTxPacketsDesc
        ch <- collector.ovsIntStatRxPacketsDesc
        ch <- collector.ovsIntStatTxDroppedDesc
        ch <- collector.ovsIntStatRxDroppedDesc
        ch <- collector.ovsIntStatTxErrorsDesc
        ch <- collector.ovsIntStatRxErrorsDesc
        ch <- collector.ovsIntStatRxCrcErrDesc
        ch <- collector.ovsIntStatRxFrameErrDesc
        ch <- collector.ovsIntStatRxOverErrDesc
}


// CSVToMap takes a reader and returns an array of dictionaries, using the header row as the keys
func CSVToMap(reader io.Reader) []map[string]string {
	r := csv.NewReader(reader)
        r.LazyQuotes = true
	rows := []map[string]string{}
	var header []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if header == nil {
			header = record
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			rows = append(rows, dict)
		}
	}
	return rows
}

func StringToFloat64(in string)  float64 {
ret,_ := strconv.ParseFloat(in, 64)
return ret
}

func StatsCSV(in string) []map[string]string {
  result := strings.Replace(in,"}","", -1)
  result2 := strings.Replace(result,"{","", -1)
  stage1 := strings.Split(result2,",")
  rows := []map[string]string{}
  arr1 := make([]string, len(stage1))
  arr2 := make([]string, len(stage1))
  for _, st1 := range stage1 {
    stage2 := strings.Split(st1,"=")
    arr1 = append(arr1,strings.TrimSpace(stage2[0]))
    if len(stage2) > 1 {
        if stage2[1] != "" {
            arr2 = append(arr2,strings.TrimSpace(stage2[1]))
       } else {
            arr2 = append(arr2,"")
       }
    }else{
      stage2 = append(stage2,"")
      arr2 = append(arr2,"")
    }
  }

   dict := map[string]string{}
   for i := range arr1 {
          dict[arr1[i]] = arr2[i]
   }


   rows = append(rows, dict)

  _, err := json.MarshalIndent(rows, "", "  ")
  if err != nil {
    fmt.Println("error:", err)
  }

  return rows
}

func BytesToString(b []byte) []string {
    bh := strings.Split(string(b),"\n")
    return bh
}

func CSVToMapS(in string) []map[string]string {
var test = CSVToMap(strings.NewReader(in))
return test
}

func LldpMetrics(collector *ovsCollector,ch chan<- prometheus.Metric) {
	cmd := exec.Command("lldpcli", "show", "neighbors", "-f", "xml")
	cmd2 := exec.Command("hostname")
	out, err := cmd.CombinedOutput()
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	var lldp Data
	err = xml.Unmarshal([]byte(out), &lldp)
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	for _, inf := range lldp.Interfaces {
		ch <- prometheus.MustNewConstMetric(
			collector.LLDPInterfaceInfo,
			prometheus.GaugeValue,
			float64(1),
			strings.TrimSuffix(string([]byte(out2)),"\n"),
			inf.Name,
			inf.Chassis.Id,
			inf.Chassis.Name,
			inf.Chassis.Desc,
			inf.Chassis.MgmtIp,
			inf.Port.Id,
			inf.Port.Desc)
	}
}

func PortMetrics(p string,collector *ovsCollector,ch chan<- prometheus.Metric) {
  metricValue := float64(1)
  hnamecmd := exec.Command("hostname")
  hnameout, herr := hnamecmd.CombinedOutput()
  if herr != nil {

  }
 ifdata, err := exec.Command("ovs-vsctl","-f","csv","--columns=_uuid,name,admin_state,link_state,link_speed,error,duplex,link_resets,mtu,mac_in_use,ofport,status,type,statistics", "list", "Interface", p ).CombinedOutput()
           if err != nil {
                  //fmt.Print("failed to get interface data: %v", err)
            }else{
              for _, element := range CSVToMapS(string(ifdata)) {
                ch <- prometheus.MustNewConstMetric(collector.ovsIntInfo,
                prometheus.CounterValue,
                metricValue,
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
                element["_uuid"],
                element["name"],
                element["admin_state"],
                element["link_state"],
                element["link_speed"],
                element["error"],
                element["duplex"],
                element["link_resets"],
                element["mtu"],
                element["mac_in_use"],
                element["ofport"],
                element["status"],
                element["type"])
                for _, element3 := range StatsCSV(string(element["status"])){
		ch <- prometheus.MustNewConstMetric(collector.ovsIntHwDesc,
		prometheus.CounterValue,
		metricValue,
		strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		element["name"],
		element3["driver_name"],
		element3["driver_version"],
		element3["firmware_version"])
                }
                for _, element2 := range StatsCSV(string(element["statistics"])){

        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatTxBytesDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["tx_bytes"]),
		"tx_bytes",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxBytesDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_bytes"]),
		"rx_bytes",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatTxPacketsDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["tx_packets"]),
		"tx_packets",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxPacketsDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_packets"]),
		"rx_packets",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatTxDroppedDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["tx_dropped"]),
		"tx_dropped",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxDroppedDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_dropped"]),
		"rx_dropped",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatTxErrorsDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["tx_errors"]),
		"tx_errors",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxErrorsDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_errors"]),
		"rx_errors",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxCrcErrDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_crc_err"]),
		"rx_crc_err",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxFrameErrDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_frame_err"]),
		"rx_frame_err",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)
        ch <- prometheus.MustNewConstMetric(
                collector.ovsIntStatRxOverErrDesc,
                prometheus.GaugeValue,
                StringToFloat64(element2["rx_over_err"]),
		"rx_over_err",
                strings.TrimSuffix(string([]byte(hnameout)),"\n"),
		p)

                 }
               }

            }
          }

func (collector *ovsCollector) Collect(ch chan<- prometheus.Metric) {
  _,err := exec.Command("ovs-vsctl","show").CombinedOutput()
  _,err2 := exec.Command("lldpcli","show","neighbors").CombinedOutput()
  metricValue := float64(1)
  hnamecmd := exec.Command("hostname")
  hnameout, herr := hnamecmd.CombinedOutput()
  if herr != nil {
  
  }
  if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
      fmt.Printf("openvswitch service not found.  Exit code: %v\n", exitErr)
    } else {
      fmt.Printf("failed to run systemctl: %v", err)
      os.Exit(1)
    }
    os.Exit(1)
  } else {
    cmd4 := "ovs-appctl bond/list | sed 's/, /:/g' | awk '{print $1,$2,$4}' | tr ' ' ','"
    bonds, err := exec.Command("bash","-c",cmd4).CombinedOutput(); if err != nil{
        fmt.Printf("failed to get bonds: %v", err)
    }else{
        for _, element := range CSVToMapS(string(bonds)) {
            ch <- prometheus.MustNewConstMetric(collector.bondInfo,
            prometheus.CounterValue,
            metricValue,
            strings.TrimSuffix(string([]byte(hnameout)),"\n"),
            element["bond"],
            element["slaves"],
            element["type"])
            sinf := strings.Split(element["slaves"],":")
            for _, s := range sinf {
               PortMetrics(s,collector,ch)
             }
            }
    }

    // Create a *ovs.Client.  Specify ovs.OptionFuncs to customize it.
    c := ovs.New(
        // Prepend "sudo" to all commands.
        ovs.Sudo(),
    )

    bridges, err := c.VSwitch.ListBridges()
    if err != nil {
            fmt.Printf("failed to get bridges: %v", err)
    }else{
 

    for _, br := range bridges {
      ports, err := c.VSwitch.ListPorts(br)
     
      if err != nil {
            fmt.Printf("failed to list ports: %v", err)
      }else{
       for _, p := range ports {
        PortMetrics(p,collector,ch)
       }
        }
      }
    }
  
    }
  if err2 != nil {
    if exitErr, ok := err2.(*exec.ExitError); ok {
      fmt.Printf("lldpd service not found.  Exit code: %v\n", exitErr)
    } else {
      fmt.Printf("failed to run systemctl: %v", err2)
      os.Exit(1)
    } 
  }else {
        LldpMetrics(collector,ch)
  }
}

func main() {

var (
app           = kingpin.New("openvswitch_exporter", "Prometheus metrics exporter for openvswitch")
listenAddress = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9700").String()
metricsPath   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
)
kingpin.MustParse(app.Parse(os.Args[1:]))


  //Create a new instance of the ovscollector and 
  //register it with the prometheus client.
  ocollector := newovsCollector()
  prometheus.MustRegister(ocollector)

http.Handle(*metricsPath, promhttp.Handler())
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`
                <html>
                <head><title>Openvswitch Exporter</title></head>
                <body>
                <h1>Openvswitch Exporter</h1>
                <p><a href='` + *metricsPath + `'>Metrics</a></p>
                </body>
                </html>`))
        })
        log.Info("Beginning to serve on port ",*listenAddress)
        log.Fatal(http.ListenAndServe(*listenAddress, nil))

  //This section will start the HTTP server and expose
  //any metrics on the /metrics endpoint.
//  http.Handle("/metrics", promhttp.Handler())
//  log.Info("Beginning to serve on port :9700")
//  log.Fatal(http.ListenAndServe(":9700", nil))
}
