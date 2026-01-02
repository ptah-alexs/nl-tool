package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	"slices"

	ii "github.com/ptah-alexs/ii-nl-lib"
)

func rmax(arg1, arg2 int64) int64 {
	if arg1 > arg2 {
		return arg1
	} else {
		return arg2
	}
}

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
	cmds := []string{"check", "export", "import", "info", "sync", "check2"}
	if !slices.Contains(cmds, args[0]) {
		fmt.Printf("Wrong cmd: %s\n", args[0])
		os.Exit(1)
	}

	nl := ii.OpenNL(*db_opt)
	now := time.Now().Unix()
	path := ""
	inactivetime := 604800

	switch cmd := args[0]; cmd {
		case "sync":
			nls := make(map[string]ii.NodeT, 10) // общий список инстансов
			ks := []string{} // список url  из nodes.txt
			for _, val := range(nl.Nodes) { // обходит список станций из своего nodes.txt
				fmt.Printf("Проверяем %s\n", val.Url)
				// ks = append(ks, val.Url)   //  добавление url станций в общий список, нужно для сохранения последовательности при записи
				tmp, ok := nls[val.Url] // проверяем наличие инстанса в общем списке
				if ok { // если ключ есть, то
					rtime := tmp.LastEx // сохраняем время последней удачной проверки инстанса
					tmp.LastEx = rmax(rtime, val.LastEx) // присваиваем наибольшее из времен удачной проверки инстанса (из вывода с дугих станций или из нашей базы)
					if now - tmp.LastEx < int64(inactivetime) || *ignore_opt { // если время последней удачной проверки инстанса меньше недели или установлен ключ для игнорирования времени на других станциях
						tmp.AltPath = true // устанавливаем признак наличия инстанса на других станциях
						tmp.Masked = false // включаем инстанс
					} else { // если время последней удачной проверки инстанса больше недели
						tmp.AltPath = false // снимаем признак наличия инстанса на других станциях
						tmp.Masked = true // выключаем инстанс
					}
				} else { // если ключа нет, то
					ks = append(ks, val.Url)   //  добавление url станций в общий список, нужно для сохранения последовательности при записи
					val.AltPath = false // снимаем признак наличия инстанса на других станциях
					tmp.Masked = false // включаем инстанс
					tmp = val //копируем информацию об инстансе
				}
				nls[val.Url] = tmp // заполняем словарь
				if val.Exclude && !*acheck_opt {continue} // если станция в исключении и не установлена опция -a, пропускаем цикл
				code, answ := ii.Getre(fmt.Sprintf("%s/nodes.txt", strings.TrimSuffix(val.Url, "/")), 2000) // пытаемся скачать nodes.txt с очередной станции
				if code != 200 {continue} // если ошибка пропускаем цикл
				if !strings.Contains(answ[0], "\t") {continue} // если в первой строке нет разделителя полей tab, значит формат не правильный, пропускаем цикл
				for _, elem := range(answ) { // обходим список инстансов в nodes.txt станции
					ns := ii.Parse(elem) // заполняем структуру информации о ноде
					fmt.Printf("Проверяем подсписок %s\n", ns.Url)
					tmp, ok := nls[ns.Url] // проверяем что этот инстанс у нас уже есть
					if ok { // если ключ есть, то
						rtime := tmp.LastEx // сохраняем время последней удачной проверки инстанса
						tmp.LastEx = rmax(rtime, ns.LastEx) // присваиваем наибольшее из времен удачной проверки инстанса (из вывода с дугих станций или из нашей базы)
						if now - tmp.LastEx < int64(inactivetime) || *ignore_opt { // если время последней удачной проверки инстанса меньше недели или установлен ключ для игнорирования времени на других станциях
							tmp.AltPath = true // устанавливаем признак наличия инстанса на других станциях
							tmp.Masked = false // включаем инстанс
						} else { // если время последней удачной проверки инстанса больше недели
							tmp.AltPath = false // снимаем признак наличия инстанса на других станциях
							tmp.Masked = true // выключаем инстанс
						}
					} else { // если ключа нет, то
						ks = append(ks, ns.Url)   //  добавление url станций в общий список, нужно для сохранения последовательности при записи
						ns.AltPath = false // снимаем признак наличия инстанса на других станциях
						tmp.Masked = false // включаем инстанс
						tmp = ns //копируем информацию об инстансе
					}
					nls[ns.Url] = tmp // заполняем словарь
				}
			}
			nl.Nodes = []ii.NodeT {}
			for _, val := range(ks) { // проходим по списку адресов станций нашей базы
				if !*skip_opt{ // если включена опция -s,
					if !ii.CheckII(nls[val].Url){ // если проверка на ii-шность провалена
						tmp := nls[val]
						tmp.Masked = true // включаем инстанс
						tmp.AltPath = false // снимаем признак наличия инстанса на других станциях
						nls[val] = tmp
					}
				}
				nl.Nodes = append(nl.Nodes, nls[val]) // добавляем инстанс из общего списка в базу
			}
			nl.Write(nl.Path) // пишем базу в файл
	case "check":
		for idx, val := range(nl.Nodes) {
			if val.Masked && !*acheck_opt {continue}
			if val.Exclude {continue}
			if ii.CheckII(val.Url) {
				nl.Nodes[idx].LastEx = now
				nl.Nodes[idx].Masked = false
			} else {
				if (now - nl.Nodes[idx].LastEx) > 604800 {nl.Nodes[idx].Masked = true}
			}
			if val.AltPath && (now - nl.Nodes[idx].LastEx) < 604800 {
				nl.Nodes[idx].Masked = false
			} else {
				nl.Nodes[idx].AltPath = false
			}
		}
		nl.Write(*db_opt)
	case "info":
		widthu, widthn := 0, 0
		for _, val := range(nl.Nodes) {
			if a:= len(val.Url); a > widthu {widthu = a}
			if b:= len(val.Name);b > widthn {widthn = b}
		}
		fmt.Printf("|     | Ссылка%s | Имя%s | Дата успешной проверки | Выкл  | Alt | Искл  |\n", strings.Repeat(" ", widthu - 6), strings.Repeat(" ", widthn - 3))
		fmt.Printf("|-----|--------%s|-----%s|------------------------|-------|-----|-------|\n", strings.Repeat("-", widthu - 6), strings.Repeat("-", widthn - 3))
		for indx, val := range(nl.Nodes) {
			rpu := widthu - len(val.Url)
			rpn := widthn - len(val.Name)
			bm, ba, be := "-", "-", "-"
			if val.Masked {bm = "+"}
			if val.AltPath {ba = "+"}
			if val.Exclude {be = "+"}
			fmt.Printf("| % 3d | %s%s | %s%s | %s    |   %s   |  %s  |   %s   |\n", indx+1, val.Url, strings.Repeat(" ", rpu), val.Name, strings.Repeat(" ", rpn), time.Unix(val.LastEx, 0).Format("2006-01-02 15:04:05"), bm, ba, be)
		}
	case "import":
		if len(args) < 2 {path = expf} else {path = args[1]}
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
		if len(args) < 2 {path = expf} else {path = args[1]}
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
	}
}
