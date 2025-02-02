package main

import (
	"context"
	"fmt"
	"inc/lib"
	"inc/lib/helpers"
	"os"
	"os/signal"
	"syscall"

	_ "inc/plugins"
	_ "inc/plugins/convert"
	_ "inc/plugins/ai"
	_ "inc/plugins/download"
	_ "inc/plugins/group"
	_ "inc/plugins/info"
	_ "inc/plugins/owner"
	_ "inc/plugins/tools"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/subosito/gotenv"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func init() {
	gotenv.Load()
	store.DeviceProps.PlatformType = waProto.DeviceProps_SAFARI.Enum()
	store.DeviceProps.Os = proto.String(os.Getenv("BOTNAME"))
}

var log helpers.Logger

func main() {
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New("sqlite3", "file:mao.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	handler := lib.NewHandler(container)
	log.Info("Connecting Socket")
	client := handler.Client()
	client.PrePairCallback = func(jid types.JID, platform, businessName string) bool {
		log.Info("Connected Socket")
		return true
	}

	if client.Store.ID == nil {
		// No ID stored, new login
		// Switch Mode
		switch int(questLogin()) {
		case 1:
			fmt.Print("Masukan Nomor (628xx) : ")
			var nomor string
			_, err := fmt.Scanln(&nomor)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			if err := client.Connect(); err != nil {
				panic(err)
			}

			code, err := client.PairPhone(nomor, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
			if err != nil {
				panic(err)
			}

			fmt.Println("Code Kamu : " + code)
			break
		case 2:
			qrChan, _ := client.GetQRChannel(context.Background())
			if err := client.Connect(); err != nil {
				panic(err)
			}
			for evt := range qrChan {
				switch string(evt.Event) {
				case "code":
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					log.Info("Qr Required")
					break
				}
			}
			break
		default:
			panic("Pilih apa?")
		}
	} else {
		// Already logged in, just connect
		if err := client.Connect(); err != nil {
			panic(err)
		}
		log.Info("Connected Socket")
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

func questLogin() int {
	fmt.Println("Silahlan Pilih Opsi Login :")
	fmt.Println("1. Pairing Code")
	fmt.Println("2. Qr")
	fmt.Print("Pilih : ")
	var input int
	_, err := fmt.Scanln(&input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 0
	}

	return input
}
