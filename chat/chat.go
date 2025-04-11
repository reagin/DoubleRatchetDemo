package chat

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/reagin/double_ratchet/core"
)

func init() {
	localPort = "9090"
	remotePort = "8080"
	localAddress = "127.0.0.1:" + localPort
	listenAddress = "0.0.0.0:" + localPort
	server = core.NewServer(listenAddress)
	client = core.NewClient(localAddress, remoteAddress+":"+remotePort)
}

func StartDoubleRatchet() {
	myApp := app.NewWithID("com.github.reagin.double_ratchet")
	myWindow := myApp.NewWindow("DoubleRatchet")
	myWindow.Resize(fyne.NewSize(860, 550))
	myWindow.SetMaster()
	myWindow.CenterOnScreen()
	myWindow.SetFixedSize(true)

	chatList := widget.NewList(
		func() int {
			return len(dataList)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template Text")
			return container.NewHBox(layout.NewSpacer(), label, layout.NewSpacer())
		},
		func(lii widget.ListItemID, co fyne.CanvasObject) {
			item := dataList[lii]
			cont := co.(*fyne.Container)

			textLabel := widget.NewLabel(string(item.Content))

			if item.IsSelf {
				cont.Objects = []fyne.CanvasObject{
					layout.NewSpacer(),
					textLabel,
				}
			} else {
				cont.Objects = []fyne.CanvasObject{
					textLabel,
					layout.NewSpacer(),
				}
			}

			cont.Refresh()
		},
	)
	// 创建聊天窗口的主容器
	chatContainer := container.NewScroll(chatList)

	// 创建输入框
	input := widget.NewMultiLineEntry()
	input.SetPlaceHolder("Type your message...")
	// 创建文件按钮
	fileButton := widget.NewButton("File", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				log.Println("❌ 选择文件失败")
				return
			}
			fileData, err := io.ReadAll(reader)
			if err != nil {
				log.Println("❌ 读取文件失败:", err)
				return
			}
			fileName := reader.URI().Name()
			fileMessage := &FileTrunk{
				FileName: fileName,
				Content:  fileData,
			}
			fileMessageBytes, _ := json.Marshal(fileMessage)
			message := &Message{FileType, false, fileMessageBytes}
			messageBytes, _ := json.Marshal(message)
			sendChannel <- messageBytes

			chatInfor := &Message{FileType, true, []byte("Send File: " + fileName)}
			dataList = append(dataList, chatInfor)
			chatList.Refresh()
		}, myWindow)
	})
	// 创建发送按钮
	sendButton := widget.NewButton("Send", func() {
		if input.Text != "" {
			content := []byte(input.Text)

			// 发送给对方的信息
			message := &Message{TextType, false, content}
			messageBytes, _ := json.Marshal(message)
			sendChannel <- messageBytes

			chatInfor := &Message{TextType, true, content}
			dataList = append(dataList, chatInfor)
			chatList.Refresh()
			input.SetText("")
		}
	})
	// 创建设置按钮
	settingButton := widget.NewButton("Setting", func() {
		remoteEntry := widget.NewEntry()
		remoteEntry.SetText(remoteAddress)

		form := dialog.NewForm(
			"Setting",
			"Save",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Remote Address", remoteEntry),
			},
			func(confirm bool) {
				if confirm {
					if remoteEntry.Text == remoteAddress {
						return
					}
					if remoteAddress != "" && remoteEntry.Text == "" {
						runMode = ServerMode
					} else {
						runMode = ClientMode
					}
					remoteAddress = remoteEntry.Text
					isChange <- true
				}
			},
			myWindow,
		)

		form.Resize(fyne.NewSize(300, 160))
		form.Show()
	})
	var buttonContainer *fyne.Container
	var buttonLayout = layout.NewGridWrapLayout(fyne.NewSize(140, 50))
	// 设置按钮容器
	buttonContainer = container.New(buttonLayout, fileButton, sendButton, settingButton)
	// 设置底部容器
	bottomContainer := container.NewBorder(nil, nil, nil, buttonContainer, input)
	// 设置主界面容器
	mainContainer := container.NewBorder(nil, bottomContainer, nil, nil, chatContainer)

	// 启动协程更新状态
	go func() {
		for {
			switch runMode {
			case ClientMode:
				client = core.NewClient(localAddress, remoteAddress+":"+remotePort)
				sendChannel = client.SendChannel
				recvChannel = client.RecvChannel
				go client.StartClient()
			case ServerMode:
				server = core.NewServer(listenAddress)
				sendChannel = server.SendChannel
				recvChannel = server.RecvChannel
				go server.StartServer()
			}
			<-isChange
			dataList = dataList[:0]
			chatList.Refresh()
		}
	}()

	// 启动协程接收信息
	go func() {
		message := &Message{}
		fileTrunk := &FileTrunk{}
		for {
			var messageBytes []byte
			select {
			case messageBytes = <-recvChannel:
			default:
				continue // 如果没有新消息，继续循环
			}

			if err := json.Unmarshal(messageBytes, message); err != nil {
				log.Fatalf("🤯 解析信息失败: %s\n", err.Error())
			}

			switch message.Type {
			case FileType:
				if err := json.Unmarshal(message.Content, fileTrunk); err != nil {
					log.Printf("❌ 解析文件数据失败: %s\n", err)
					continue
				}

				// 更新 UI 显示
				chatInfor := &Message{FileType, message.IsSelf, []byte("Received File: " + fileTrunk.FileName)}
				dataList = append(dataList, chatInfor)
				chatList.Refresh()

				savePath := "./received_" + fileTrunk.FileName
				if err := os.WriteFile(savePath, fileTrunk.Content, 0644); err != nil {
					log.Printf("❌ 保存文件失败: %s\n", err)
					continue
				}
				log.Printf("✅ 文件已保存: %s\n", savePath)
			case TextType:
				dataList = append(dataList, message)
				chatList.Refresh()
			}
		}
	}()

	myWindow.SetContent(mainContainer)
	myWindow.ShowAndRun()
}
