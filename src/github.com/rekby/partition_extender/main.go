package partitionextender
import (
	"fmt"
	"os"
	"unicode"
)

func Main(){
	if len(os.Args) < 3 || os.Args[1] == "--help" {
		printUsage()
		return
	}

	path := os.Args[1]
	if len(path) < 2 || !unicode.IsDigit(rune( path[len(path)-1])) || unicode.IsDigit(rune( path[len(path)-2])) {
		fmt.Println("ERR\nBad partition number")
		printUsage()
		return
	}

	stat, err := os.Stat(path)
	if err != nil {
		fmt.Println("ERR\nCan't stat partition path")
	}

}

func printUsage(){
	fmt.Printf(`%s <device> <partnumber> [+SIZE]
<devide> - full path for file of device, which need to extend, for example /dev/sda or /dev/hdd
<partnumber> - number of partition: 1,2,3 or 4
[+SIZE] - optional parametr, set size for growing in GB. Default partition will extend to max size.

return codes:
0 - OK, partition was extended
1 - partition is max size already. It state can return when autoresize only.
2 - partition can't be extend

stdout:
OK\n - partition was extended
ALREADY_MAX\n - partition already have max size and can't grow. It state can return when autoresize only.
ERR\nsome message - partition can't be extended

example usages:
partextender /dev/sda 2
partextender /dev/sda 2 +10
`, os.Args[0])
}