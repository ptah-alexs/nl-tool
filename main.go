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
		now := time.Now().Unix()
		for idx, val := range(nl.Nodes) {
			if val.Masked && !*acheck_opt {continue}
			if val.Exclude {continue}
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
		ks := []string{} // список url  из nodes.txt
		change := false // признак изменений в списке
		now := time.Now().Unix()
		var gtime int64 = 0
		for _, val := range(nl.Nodes) { // обходит список станций из nodes.txt
			ks = append(ks, val.Url)   //  добавление url станции в общий список
			if val.Exclude && !*acheck_opt {continue} // если станция в исключении и не установлена опция -a, пропускаем цикл
			code, answ := ii.Getre(fmt.Sprintf("%s/nodes.txt", strings.TrimSuffix(val.Url, "/")), 2000) // пытаемся скачать nodes.txt с очередной станции
			if code != 200 {continue} // если ошибка пропускаем цикл
			if !strings.Contains(answ[0], "\t") {continue} // если в первой строке нет разделителя полей tab, значит формат не правильный, пропускаем цикл
			for _, elem := range(answ) { // обходим список инстансов в nodes.txt станции
				ns := ii.Parse(elem) // заполняем структуру информации о ноде
				tmp, ok := nls[ns.Url] // проверяем что этот инстанс у нас уже есть
				if !ok { // если нет,
					nls[ns.Url] = ns // добавляем в общий список
				} else { // если есть,
					if ns.LastEx > tmp.LastEx { // проверяем что время последней проверки из nodes.txt станции больше чем сохранённое у нас в общем списке
						tmp.LastEx = ns.LastEx // обновляем поле структуры
						nls[ns.Url] = tmp // записываем изменения в общий список
					}
				}
			}
		}
		for idx, val := range(ks) { // проходим по списку адресов станций нашей базы
			tmp, ok := nls[val] // загружаем структуру для работы
			if ok { // проверка что по данному ключу есть данные
				change = false // устанавливаем признак изменений в выкл
				if tmp.LastEx > nl.Nodes[idx].LastEx { // если время последнего обновления в nodes.txt больше чем у нас в базе
					change = true // устанавливаем признак изменений во вкл
					gtime = tmp.LastEx // запоминаем время изменения
				} else if nl.Nodes[idx].LastEx - tmp.LastEx < 604800 { // если у нас в базе время обновления больше чем в общем списке, но разница меньше чем 7 суток
					change = true // устанавливаем признак изменений во вкл
					gtime = nl.Nodes[idx].LastEx // запоминаем время изменения
				}
				if *ignore_opt {change = true} // если установлена опция -a, устанавливаем признак изменений во вкл
				if now - gtime > 604800 { // если самое больше время старше чем текущее время на неделю, то
					change = false // устанавливаем признак изменений в выкл
					nl.Nodes[idx].AltPath = false // убираем признак наличия инстанса на других станциях
					nl.Nodes[idx].Masked = true // выключаем инстанс
				}
				if change { // если признак изменений вкл
					nl.Nodes[idx].AltPath = true // устанавливаем признак того что инстанс есть в списках других станций
					nl.Nodes[idx].Masked = false // делаем инстанс видимым в нашей базе
				}
				delete(nls, val) // удаляем инстанс из общего списка, потому что он уже есть в нашей базе
			}
		}
		for _, val := range(nls) { //проходим по оставшемуся списку инстансов
			if *skip_opt { // если включена опция -s,
				nl.Nodes = append(nl.Nodes, val) // то пропускаем проверку на ii-шность, и добавляем инстанс из общего списка в базу
			} else if ii.CheckII(val.Url){ // в противном случае, проверяем что инстанс является станцией ii, тогда
				nl.Nodes = append(nl.Nodes, val) // добавляем инстанс из общего списка в базу
			}
		}
		nl.Write(nl.Path) // пишем базу в файл
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
