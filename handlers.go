package main

import (
    "fmt"
    "net/http"
    "os/exec"
    "regexp"
    "strings"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//the passive TCP port where OVS entries are listening
//for OpenFlow commands
var ovsPort string = "16633"

func GetMetrics(w http.ResponseWriter, r *http.Request) {
    ovsIP := r.URL.Query()["target"][0]
    
    if ovsIP == "" {
    	fmt.Fprintln(w, "Bad request!\nCorrect format is: http://<IP>:<Port>/flows?tartget=<targetIP>")
    }


    //Get all services from Kubernetes API
    services, err := clientset.CoreV1().Services("").List(metav1.ListOptions{})
    if err != nil {
        panic(err.Error())
    }
    
    //Get all nodes from Kubernetes API
    nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
	}

    //get info on all pods in the cluster
    pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
    if err != nil {
        panic(err.Error())
    }
    
    //Flow specific staticstics
    //
    //creating ovs-ofctl command for flow staticstics
    cmd := exec.Command("ovs-ofctl", "-O", "openflow13", "dump-flows", "tcp:" + ovsIP + ":" + ovsPort)
    out, err := cmd.Output()
	outString := string(out)
	//if error was occured we return
	if err != nil {
		fmt.Fprintln(w, "Error is: ", err, "\nOutput was:", outString)
		return
	}
    //if command was succesfull we further parse the output
    
    
    lines := strings.Split(outString, "\n")
    //skip the first and last lines, since it is just a response header and an empty line
    lines = lines[1:(len(lines)-1)]
    flowEntries := make([]Flow, len(lines))
    for i, entry := range lines {
    	re := regexp.MustCompile("")
    	
    	//Cookie
    	re = regexp.MustCompile("cookie=(.*?),")
    	subMatch := re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Cookie = subMatch[1]
    	}

    	//Duration
    	re = regexp.MustCompile("duration=(.*?)s,")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Duration = subMatch[1]
    	}

    	//Table
    	re = regexp.MustCompile("table=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Table = subMatch[1]
    	}

    	//Packets
    	re = regexp.MustCompile("packets=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Packets = subMatch[1]
    	}
    	
    	//Bytes
    	re = regexp.MustCompile("bytes=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Bytes = subMatch[1]
    	}
    	
    	//Idle Timeout
    	re = regexp.MustCompile("idle_timeout=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].IdleTimeout = subMatch[1]
    	}

    	//Idle Age
    	re = regexp.MustCompile("idle_age=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].IdleAge = subMatch[1]
    	}
    	    	
    	//Priority & Match rule
    	re = regexp.MustCompile("priority=(.*?),(.*?) ")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 2 {
    		flowEntries[i].Priority = subMatch[1]
    		flowEntries[i].Match    = subMatch[2]
    	}

    	//Action
    	re = regexp.MustCompile("actions=(.*)")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		flowEntries[i].Action = subMatch[1]
    	}
    	
    }
    
    //Interating through the flows to add some extra information based on Service and Node information
    for i,_ := range flowEntries {
    
        //Cheking for ServiceIPs
        for _, service := range services.Items {
	        if strings.Contains(flowEntries[i].Action, service.Spec.ClusterIP) {
                flowEntries[i].ServiceName = service.GetName()
                flowEntries[i].ServiceNamespace = service.GetNamespace()
                flowEntries[i].ServiceIP = service.Spec.ClusterIP
            }

	        if strings.Contains(flowEntries[i].Match, service.Spec.ClusterIP) {
                flowEntries[i].ServiceName = service.GetName()
                flowEntries[i].ServiceNamespace = service.GetNamespace()
                flowEntries[i].ServiceIP = service.Spec.ClusterIP
            }

        }
        
        //Cheking for NodePOD CIDRs
        for _, node := range nodes.Items {
            //this code should work if we ever start to add PodCIDR information to Kubernetes API
            //if strings.Contains(flowEntries[i].Match, node.Spec.PodCIDR) {
            //    flowEntries[i].ServiceName = node.GetName()
            //    flowEntries[i].ServiceIP = node.Status.Addresses
            //}

            podCidr := "10.244."
            re := regexp.MustCompile("[0-9]+.[0-9]+.[0-9]+.([0-9]+)")
            subMatch := re.FindStringSubmatch(node.Status.Addresses[0].Address)
            if len(subMatch) > 1 {
                podCidr += subMatch[1]
            }
            podCidr += ".0/24"
            if strings.Contains(flowEntries[i].Match, podCidr) {
                flowEntries[i].NodeName = node.GetName()
                flowEntries[i].NodeIP = node.Status.Addresses[0].Address
            }
        }
    }
    
    //Creating Prometheus compatible output for:
    //	- number of packets as "flowPackets" type Counter
    //	- number of bytes as "flowBytes" type Counter
    //	- age of the flow as "flowAge" type Gauge
    //	- idle time as "flowIdleTime" type Gauge
    
    //flowPackets
    fmt.Fprintln(w, "# HELP flowPackets The number of packets matched for the given OpenFlow entry")
    fmt.Fprintln(w, "# TYPE flowPackets counter")
    for _,entry := range flowEntries {
    	fmt.Fprintln(w, 
    		"flowPackets{match=\""  + entry.Match + 
    		"\",action=\""	 		+ entry.Action +
    		"\",table=\"" 			+ entry.Table +
    		"\",priority=\""		+ entry.Priority + 
                                      extraInfo(entry) +
    		"\"} "					+ entry.Packets)    		 
    }
     
    //flowBytes
    fmt.Fprintln(w, "# HELP flowBytes The number of bytes matched for the given OpenFlow entry")
    fmt.Fprintln(w, "# TYPE flowBytes counter")
    for _,entry := range flowEntries {
    	fmt.Fprintln(w, 
    		"flowBytes{match=\"" 	+ entry.Match + 
    		"\",action=\""	 		+ entry.Action +
    		"\",table=\"" 			+ entry.Table +
    		"\",priority=\""		+ entry.Priority +
                                      extraInfo(entry) +
    		"\"} "					+ entry.Bytes)    		 
    }
    
    //flowAge
    fmt.Fprintln(w, "# HELP flowAge The number of seconds have passed since the given OpenFlow entry was created")
    fmt.Fprintln(w, "# TYPE flowAge gauge")
    for _,entry := range flowEntries {
    	fmt.Fprintln(w, 
    		"flowAge{match=\""	 	+ entry.Match + 
    		"\",action=\""	 		+ entry.Action +
    		"\",table=\"" 			+ entry.Table +
    		"\",priority=\""		+ entry.Priority +
                                      extraInfo(entry) +
    		"\"} "					+ entry.Duration)    		 
    }
    
    //flowIdleTime
    //currently this is not working since the ovs-ofctl -O openflow13 will not get this stat back
    //fmt.Fprintln(w, "# HELP flowIdleTime The number of seconds have passed since the last packet has seen for the given OpenFlow entry")
    //fmt.Fprintln(w, "# TYPE flowIdleTime gauge")
    //for _,entry := range flowEntries {
    //	fmt.Fprintln(w, 
    //		"flowIdleTime{match=\""	+ entry.Match + 
    //		"\",action=\""	 		+ entry.Action +
    //		"\",table=\"" 			+ entry.Table +
    //		"\",priority=\""		+ entry.Priority +
    //                                  extraInfo(entry) +
    //		"\"} "					+ entry.IdleAge)    		 
    //}
    
    //Port specific staticstics
    //
    //Creating ovs-ofctl command for port statisctics
    cmd = exec.Command("ovs-ofctl", "dump-ports", "tcp:" + ovsIP + ":" + ovsPort)
    out, err = cmd.Output()
	outString = string(out)
	//if error was occured we return
	if err != nil {
		fmt.Fprintln(w, "Error is: ", err, "\nOutput was:", outString, "\nOVS IP is: ", ovsIP)
		return
	}
    //if command was succesfull we further parse the output
    
    
    lines = strings.Split(outString, "\n")
    //skip the first and last lines, since it is just a response header and an empty line
    lines = lines[1:(len(lines)-1)]
    portEntries := make([]Port, int(len(lines)/2))
    for i:=0; i<len(lines); i+=2 {
    	twoLines := lines[i] + lines[i+1]
    	
    	//Searching every entry in one line as the following (there is no new line charachter between them):
    	//  port 1: rx pkts=1148284, bytes=76073652, drop=0, errs=0, frame=0, over=0, crc=0
        //          tx pkts=1814122, bytes=90439143776, drop=0, errs=0, coll=0
    	re := regexp.MustCompile("port +(.*?): rx pkts=(.*?), bytes=(.*?), drop=(.*?), errs=(.*?), frame=(.*?), over=(.*?), crc=(.*?) .*tx pkts=(.*?), bytes=(.*?), drop=(.*?), errs=(.*?), coll=(.*)")
    	subMatch := re.FindStringSubmatch(twoLines)
    	if len(subMatch) > 13 {
    		portEntries[int(i/2)].PortNumber   = noQuestionMark(subMatch[1])
			portEntries[int(i/2)].RxPackets    = noQuestionMark(subMatch[2])
			portEntries[int(i/2)].RxBytes      = noQuestionMark(subMatch[3])
			portEntries[int(i/2)].RxDrops      = noQuestionMark(subMatch[4])
			portEntries[int(i/2)].RxErrors     = noQuestionMark(subMatch[5])
			portEntries[int(i/2)].RxFrameErr   = noQuestionMark(subMatch[6])
			portEntries[int(i/2)].RxOverruns   = noQuestionMark(subMatch[7])
			portEntries[int(i/2)].RxCrcErrors  = noQuestionMark(subMatch[8])
			portEntries[int(i/2)].TxPackets    = noQuestionMark(subMatch[9])
			portEntries[int(i/2)].TxBytes      = noQuestionMark(subMatch[10])
			portEntries[int(i/2)].TxDrops      = noQuestionMark(subMatch[11])
			portEntries[int(i/2)].TxErrors     = noQuestionMark(subMatch[12])
			portEntries[int(i/2)].TxCollisions = noQuestionMark(subMatch[13])

            for _, pod := range pods.Items {
                if (pod.Status.HostIP == ovsIP) {
                    octets := strings.Split(pod.Status.PodIP, ".")
                    if (octets[len(octets)-1] == subMatch[1]) {
                        portEntries[int(i/2)].PodName = pod.GetName()
                        portEntries[int(i/2)].PodNamespace = pod.GetNamespace()
                        portEntries[int(i/2)].PodIP = pod.Status.PodIP
                    }
                }                
            }
            
            if (subMatch[1] == "LOCAL") {
                portEntries[int(i/2)].PodName = "ToKernel"
                portEntries[int(i/2)].PodNamespace = "HostNetworking"
                portEntries[int(i/2)].PodIP = ovsIP
            }
    	} else {
    		fmt.Fprintln(w, "Output is: ", subMatch, twoLines)
    		return
    	}
    }
           
    //Creating Prometheus compatible output for every stat with portNumber identifyer:
    //	- number of packets recieved by the given OpenFlow port as "portRxPackets" type Counter
    //	- number of packets sent by the given OpenFlow port as "portTxPackets" type Counter
    //	- number of bytes recieved by the given OpenFlow port as "portRxBytes" type Counter
    //	- number of bytes sent by the given OpenFlow port as "portTxBytes" type Counter
    //	- number of packet drops in recieve side by the given OpenFlow port as "portRxDrops" type Counter
    //	- number of packet drops in sending side by the given OpenFlow port as "portTxDrops" type Counter

    //portRxPackets
    fmt.Fprintln(w, "# HELP portRxPackets The number of packet that was recieved by a given port")
    fmt.Fprintln(w, "# TYPE portRxPackets counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portRxPackets{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					      + entry.RxPackets)    		 
    }
 
    //portTxPackets
    fmt.Fprintln(w, "# HELP portTxPackets The number of packet that was sent by a given port")
    fmt.Fprintln(w, "# TYPE portTxPackets counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portTxPackets{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					      + entry.TxPackets)    		 
    }

    //portRxBytes
    fmt.Fprintln(w, "# HELP portRxBytes The number of bytes that was recieved by a given port")
    fmt.Fprintln(w, "# TYPE portRxBytes counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portRxBytes{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					    + entry.RxBytes)    		 
    }
 
    //portTxBytes
    fmt.Fprintln(w, "# HELP portTxBytes The number of bytes that was sent by a given port")
    fmt.Fprintln(w, "# TYPE portTxBytes counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portTxBytes{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					    + entry.TxBytes)    		 
    }

    //portRxDrops
    fmt.Fprintln(w, "# HELP portRxDrops The number of packets that was dropped on receive side by a given port")
    fmt.Fprintln(w, "# TYPE portRxDrops counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portRxDrops{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					    + entry.RxDrops)    		 
    }
 
    //portTxDrops
    fmt.Fprintln(w, "# HELP portTxDrops The number of packets that was dropped on sending side by a given port")
    fmt.Fprintln(w, "# TYPE portTxDrops counter")
    for _,entry := range portEntries {
    	fmt.Fprintln(w, 
    		"portTxDrops{portNumber=\"" + entry.PortNumber + "\",podName=\"" + entry.PodName + "\",podNamespace=\"" + entry.PodNamespace + "\",podIP=\"" + entry.PodIP +
    		"\"} "					    + entry.TxDrops)    		 
    }

    //Group specific statistics
    //
    //creating ovs-ofctl command for groups
    cmd = exec.Command("ovs-ofctl", "-O", "openflow13", "dump-groups", "tcp:" + ovsIP + ":" + ovsPort)
    out, err = cmd.Output()
	outString = string(out)
	//if error was occured we return
	if err != nil {
		fmt.Fprintln(w, "Error is: ", err, "\nOutput was:", outString)
		return
	}
    //if command was succesfull we further parse the output
    
    lines = strings.Split(outString, "\n")
    //skip the first and last lines, since it is just a response header and an empty line
    lines = lines[1:(len(lines)-1)]
    groupEntries := make([]Group, len(lines))
    for i, entry := range lines {
    	re := regexp.MustCompile("")
    	
    	//Group Type
    	re = regexp.MustCompile("group_id=(.*?),")
    	subMatch := re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		groupEntries[i].GroupId = subMatch[1]
    	}

    	//Group Type
    	re = regexp.MustCompile("type=(.*?),")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		groupEntries[i].GroupType = subMatch[1]
    	}
    	
    	//Split the group line into buckets
    	buckets := strings.Split(entry, "bucket=")
    	bucketEntries := make([]Bucket, len(buckets)-1)
    	for j:=1; j<len(buckets); j++ {
			re = regexp.MustCompile("actions=(.*?),?$")
			subMatch = re.FindStringSubmatch(buckets[j])
			if len(subMatch) > 1 {
				bucketEntries[j-1].Actions = subMatch[1]
			}	

            //iterate through Kubernetes PODs to find the POD backend that this bucket routes
            for _, pod := range pods.Items {
            	//the "->" is recquired so that an IP like "10.244.0.2" does not match to "10.244.0.20"
                if (strings.Contains(buckets[j], pod.Status.PodIP + "->")) {
                    bucketEntries[j-1].PodName = pod.GetName()
                    bucketEntries[j-1].PodNamespace = pod.GetNamespace()
                    bucketEntries[j-1].PodIP = pod.Status.PodIP
                }                
            }
    	}    	
    	groupEntries[i].Buckets = bucketEntries
  
		//iterate through Flow Entries to find which flow service this group rule belongs
		for _, entry := range flowEntries {
			if (strings.Contains(entry.Action, "group:" + groupEntries[i].GroupId)) {
				groupEntries[i].ServiceName = entry.ServiceName
				groupEntries[i].ServiceNamespace = entry.ServiceNamespace
				groupEntries[i].ServiceIP = entry.ServiceIP
			}
		}
	}
	

    //creating ovs-ofctl command for group-stats
    cmd = exec.Command("ovs-ofctl", "-O", "openflow13", "dump-group-stats", "tcp:" + ovsIP + ":" + ovsPort)
    out, err = cmd.Output()
	outString = string(out)
	//if error was occured we return
	if err != nil {
		fmt.Fprintln(w, "Error is: ", err, "\nOutput was:", outString)
		return
	}
    //if command was succesfull we further parse the output
    
    lines = strings.Split(outString, "\n")
    //skip the first and last lines, since it is just a response header and an empty line
    lines = lines[1:(len(lines)-1)]
    for _, entry := range lines {
    	re := regexp.MustCompile("")
    	
    	groupIndex := -1
    	
    	//Get the matching Group ID
    	re = regexp.MustCompile("group_id=(.*?),")
    	subMatch := re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		for j, group := range groupEntries {
    			if group.GroupId == subMatch[1] {
    				groupIndex = j 
    			}
    		}
    	}

    	//Duration
    	re = regexp.MustCompile("duration=(.*?)s,")
    	subMatch = re.FindStringSubmatch(entry)
    	if len(subMatch) > 1 {
    		groupEntries[groupIndex].Duration = subMatch[1]
    	}
    	
    	//Bucket byte and packet stat
    	buckets := strings.Split(entry, ":")
    	//The 0th element in this split should contain the aggregated packet/byte counter for the whole group
		re = regexp.MustCompile("packet_count=([0-9]+)")
		subMatch = re.FindStringSubmatch(buckets[0])
		if len(subMatch) > 1 {
			groupEntries[groupIndex].Packets = subMatch[1]
		}

		re = regexp.MustCompile("byte_count=([0-9]+)")
		subMatch = re.FindStringSubmatch(buckets[0])
		if len(subMatch) > 1 {
			groupEntries[groupIndex].Bytes = subMatch[1]
		}
    	 
    	//The others should contain bucket data
    	for j:=1; j<len(buckets); j++ {
			re = regexp.MustCompile("packet_count=([0-9]+)")
			subMatch = re.FindStringSubmatch(buckets[j])
			if len(subMatch) > 1 {
				groupEntries[groupIndex].Buckets[j-1].Packets = subMatch[1]
			}

			re = regexp.MustCompile("byte_count=([0-9]+)")
			subMatch = re.FindStringSubmatch(buckets[j])
			if len(subMatch) > 1 {
				groupEntries[groupIndex].Buckets[j-1].Bytes = subMatch[1]
			}
    	}    	
	}

    //Creating Prometheus compatible output for every group stat with groupId label:
    //	- number of packets that was forwarded by a group rule as "groupPackets" type Counter
    //	- number of bytes that was forwarded by a group rule as "groupBytes" type Counter
    //	- number of second that passed since a group rule was added as "groupPackets" type Gauge
    //	- number of packets that was forwarded by a bucket in a group rule as "groupBucketPackets" type Counter
    //	- number of bytes that was forwarded by a bucket in a group rule as "groupBucketBytes" type Counter
    
    //groupPackets
    fmt.Fprintln(w, "# HELP groupPackets The number of packet that was sent by a given group")
    fmt.Fprintln(w, "# TYPE groupPackets counter")
    for _,entry := range groupEntries {
    	fmt.Fprintln(w,
			"groupPackets{groupId=\"" + entry.GroupId +
			"\",groupType=\"" 		  + entry.GroupType +
			"\",serviceName=\"" 	  + entry.ServiceName +
			"\",serviceNamespace=\""  + entry.ServiceNamespace +
			"\",serviceIP=\"" 	      + entry.ServiceIP +
			"\"} "					  + entry.Packets)    		 
	}

    //groupBytes
    fmt.Fprintln(w, "# HELP groupBytes The number of bytes that was sent by a given group")
    fmt.Fprintln(w, "# TYPE groupBytes counter")
    for _,entry := range groupEntries {
    	fmt.Fprintln(w,
			"groupBytes{groupId=\""   + entry.GroupId +
			"\",groupType=\"" 		  + entry.GroupType +
			"\",serviceName=\"" 	  + entry.ServiceName +
			"\",serviceNamespace=\""  + entry.ServiceNamespace +
			"\",serviceIP=\"" 	      + entry.ServiceIP +
			"\"} "					  + entry.Bytes)    		 
	}
	
    //groupDuration
    fmt.Fprintln(w, "# HELP groupDuration The number of seconds passed since the group entry was added")
    fmt.Fprintln(w, "# TYPE groupDuration gauge")
    for _,entry := range groupEntries {
    	fmt.Fprintln(w,
			"groupDuration{groupId=\""+ entry.GroupId +
			"\",groupType=\"" 		  + entry.GroupType +
			"\",serviceName=\"" 	  + entry.ServiceName +
			"\",serviceNamespace=\""  + entry.ServiceNamespace +
			"\",serviceIP=\"" 	      + entry.ServiceIP +
			"\"} "					  + entry.Duration)    		 
	}	

    //groupBucketPackets
    fmt.Fprintln(w, "# HELP groupBucketPackets The number of packet that was sent by a given group bucket")
    fmt.Fprintln(w, "# TYPE groupBucketPackets counter")
    for _,entry := range groupEntries {
	    for _,bucket := range entry.Buckets {
	    	fmt.Fprintln(w,
				"groupBucketPackets{groupId=\""  + entry.GroupId +
				"\",groupType=\"" 				 + entry.GroupType +
				"\",serviceName=\"" 	         + entry.ServiceName +
				"\",serviceNamespace=\""         + entry.ServiceNamespace +
				"\",serviceIP=\"" 	             + entry.ServiceIP +
				"\",bucketActions=\"" 			 + bucket.Actions +
				"\",podName=\""		 			 + bucket.PodName +
				"\",podNamespace=\"" 			 + bucket.PodNamespace +
				"\",podIP=\""		 			 + bucket.PodIP +
				"\"} "					         + bucket.Packets)    		 
	    }
	}

    //groupBucketBytes
    fmt.Fprintln(w, "# HELP groupBucketBytes The number of bytes that was sent by a given group bucket")
    fmt.Fprintln(w, "# TYPE groupBucketBytes counter")
    for _,entry := range groupEntries {
	    for _,bucket := range entry.Buckets {
	    	fmt.Fprintln(w,
				"groupBucketBytes{groupId=\""    + entry.GroupId +
				"\",groupType=\"" 				 + entry.GroupType +
				"\",serviceName=\"" 	         + entry.ServiceName +
				"\",serviceNamespace=\""         + entry.ServiceNamespace +
				"\",serviceIP=\"" 	             + entry.ServiceIP +
				"\",bucketActions=\"" 			 + bucket.Actions +
				"\",podName=\""		 			 + bucket.PodName +
				"\",podNamespace=\"" 			 + bucket.PodNamespace +
				"\",podIP=\""		 			 + bucket.PodIP +
				"\"} "					         + bucket.Bytes)    		 
	    }
	}

}

func noQuestionMark(s string) string {
	if s == "?" {
		return "0"
	}
	return s
}

func extraInfo(entry Flow) string {
    var info string
    if entry.ServiceName != "" {
        info = "\",serviceName=\"" + entry.ServiceName + "\",serviceNamespace=\"" + entry.ServiceNamespace + "\",serviceIP=\"" + entry.ServiceIP  
    }
    if entry.NodeName != "" {
        info = "\",nodeName=\"" + entry.NodeName + "\",nodeIP=\"" + entry.NodeIP  
    }        
    return info
}

