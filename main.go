package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	ii "github.com/ptah-alexs/ii-nl-lib"
)

func main() {
	expf := "nodes.txt"

	db_opt := flag.String("db", "./stations.txt", "Database path (directory)")
	acheck_opt := flag.Bool("a", false, "check, sync: process all entries, without excludes")
	skip_opt := flag.Bool("s", false, "sync: skip check ii-compatibility for entries")
	ignore_opt := flag.Bool("i", false, "sync: ignore timestamp for altpath stations")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf(`Help: %s [options] command [arguments]
Commands:
	check                         - check availability station from nodelist
	export [file]                 - export nodes to file (default: nodes.txt)
	import [file]                 - import nodes from file (default: nodes.txt)
	info                          - print actual nodelist
	sync                          - sync nodelists from other station

Options:
	-db=<path>                    - database path (default: station.txt)
	-a                            - check, sync: process all entries, without excludes
	-i                            - sync: ignore timestamp for altpath stations
	-s                            - sync: skip check ii-compatibility for entries
`, os.Args[0])
		os.Exit(1)
	}
	switch cmd := args[0]; cmd {
	case "check":
		nl := ii.OpenNL(*db_opt)
		for idx, val := range(nl.Nodes) {
			if val.Masked && !*acheck_opt {continue}
			if val.Exclude {continue}
			now := time.Now().Unix()
			if ii.CheckII(val.Url) {
				nl.Nodes[idx].LastEx = now
				nl.Nodes[idx].Masked = false
			} else {
				if (now - nl.Nodes[idx].LastEx) > 604800 {nl.Nodes[idx].Masked = true}
			}
			if val.AltPath {nl.Nodes[idx].Masked = false}
		}
		nl.Write(*db_opt)
	case "info":
		nl := ii.OpenNL(*db_opt)
		widthu, widthn := 0, 0
		for _, val := range(nl.Nodes) {
			if a:= len(val.Url); a > widthu {widthu = a}
			if b:= len(val.Name);b > widthn {widthn = b}
		}
		fmt.Printf("|    | Ссылка%s | Имя%s | Дата успешной проверки | Выкл  | Alt | Искл  |\n", strings.Repeat(" ", widthu - 6), strings.Repeat(" ", widthn - 3))
		fmt.Printf("|----|--------%s|-----%s|------------------------|-------|-----|-------|\n", strings.Repeat("-", widthu - 6), strings.Repeat("-", widthn - 3))
		for indx, val := range(nl.Nodes) {
			rpu := widthu - len(val.Url)
			rpn := widthn - len(val.Name)
			bm, ba, be := "-", "-", "-"
			if val.Masked {bm = "+"}
			if val.AltPath {ba = "+"}
			if val.Exclude {be = "+"}
			fmt.Printf("| % 2d | %s%s | %s%s | %s    |   %s   |  %s  |   %s   |\n", indx+1, val.Url, strings.Repeat(" ", rpu), val.Name, strings.Repeat(" ", rpn), time.Unix(val.LastEx, 0).Format("2006-01-02 15:04:05"), bm, ba, be)
		}
	case "sync":
		nl := ii.OpenNL(*db_opt)
		nls := make(map[string]ii.NodeT, 10)
		ks := []string{}
		for _, val := range(nl.Nodes) {
			ks = append(ks, val.Url)
			if val.Exclude && !*acheck_opt {continue}
			code, answ := ii.Getre(fmt.Sprintf("%s/nodes.txt", strings.TrimSuffix(val.Url, "/")), 2000)
			if code != 200 {continue}
			if !strings.Contains(answ[0], "\t") {continue}
			for _, elem := range(answ) {
				ns := ii.Parse(elem)
				tmp, ok := nls[ns.Url]
				if !ok {
					nls[ns.Url] = ns
				} else {
					if ns.LastEx > tmp.LastEx {
						tmp.LastEx = ns.LastEx
						nls[ns.Url] = tmp
					}
				}
			}
		}
		for idx, val := range(ks) {
			tmp, ok := nls[val]
			if ok {
				change := false
				if tmp.LastEx > nl.Nodes[idx].LastEx {
					change = true
				} else if nl.Nodes[idx].LastEx - tmp.LastEx < 604800 {
					change = true
				}
				if *ignore_opt {change = true}
				if change {
					nl.Nodes[idx].AltPath = true
					nl.Nodes[idx].Masked = false
				}
				delete(nls, val)
			}
		}
		for _, val := range(nls) {
			if *skip_opt {
				nl.Nodes = append(nl.Nodes, val)
			} else if ii.CheckII(val.Url){
				nl.Nodes = append(nl.Nodes, val)
			}
		}
		nl.Write(nl.Path)
	case "import":
		path := ""
		if len(args) < 2 {path = expf} else {path = args[1]}
		nl := ii.OpenNL(*db_opt)
		fd, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error read file")
			os.Exit(1)
		}
		defer fd.Close()
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			if scanner.Text() == "" {continue}
			aa := strings.Split(scanner.Text(), "\t")
			et := []string{"", "", "0", "-"}
			for idx, val := range(aa) {
				et[idx] = val
				if idx == 3 {break}
			}
			sd := false
			for _, val := range(nl.Nodes) {
				if et[0] == val.Url {
					sd = true
					break
				}
			}
			var le int64 = 0
			fmt.Sscan(et[2], &le)
			if !sd {nl.Nodes = append(nl.Nodes, ii.NodeT{Url: et[0], Name: et[1], Masked: false, AltPath: false, LastEx: le, Exclude: false})}
		}
		nl.Write(nl.Path)
	case "export":
		path := ""
		if len(args) < 2 {path = expf} else {path = args[1]}
		nl := ii.OpenNL(*db_opt)
		fd, err := os.Create(path)
		if err != nil {
			fmt.Printf("Error open file")
			os.Exit(1)
		}
		buffer := bufio.NewWriter(fd)
		 _, err = buffer.WriteString(ii.Generate(&nl.Nodes))
		if err != nil {
			fmt.Printf("Error write file")
			os.Exit(1)
		}
		if err := buffer.Flush(); err != nil {
			fmt.Printf("Error write file")
			os.Exit(1)
		}
	default:
		fmt.Printf("Wrong cmd: %s\n", cmd)
		os.Exit(1)
	}
}
