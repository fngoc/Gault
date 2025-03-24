package client

import (
	pb "Gault/api/pb/api/proto"
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/grpc/metadata"
)

// showLoginMenu экран логина/регистрации
func showLoginMenu(app *tview.Application) tview.Primitive {
	message := tview.NewTextView().
		SetText("Use [Tab] to switch fields").
		SetTextAlign(tview.AlignCenter)

	loginField := tview.NewInputField().
		SetLabel("Login: ").
		SetFieldWidth(40)

	passField := tview.NewInputField().
		SetLabel("Password: ").
		SetMaskCharacter('*').
		SetFieldWidth(40)

	form := tview.NewForm().
		AddFormItem(loginField).
		AddFormItem(passField).
		AddButton("Login", func() {
			login := loginField.GetText()
			pass := passField.GetText()

			response, err := autClient.Login(
				context.Background(),
				&pb.LoginRequest{
					Login:    login,
					Password: pass,
				},
			)
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Login error: %v", err))
				return
			}

			message.SetTextColor(tcell.ColorGreen).SetText(fmt.Sprintf("Login successful!"))
			showDataScreen(app, response.UserUid, response.Token, message)
		}).
		AddButton("Register", func() {
			login := loginField.GetText()
			pass := passField.GetText()

			response, err := autClient.Registration(
				context.Background(),
				&pb.RegistrationRequest{
					Login:    login,
					Password: pass,
				},
			)
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Registration error: %v", err))
				return
			}

			message.SetTextColor(tcell.ColorGreen).SetText(fmt.Sprintf("Registration successful!"))
			showDataScreen(app, response.UserUid, response.Token, message)
		}).
		AddButton("Exit", func() {
			app.Stop()
		})

	form.SetBorder(true).
		SetTitle(" Login Menu ").
		SetTitleAlign(tview.AlignCenter)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(message, 1, 1, false)

	return flex
}

// showDataScreen экран с таблицей данных и кнопками добавления/чтения/скачивания данных
func showDataScreen(app *tview.Application, userUID, token string, message *tview.TextView) {
	table := tview.NewTable()
	form := tview.NewForm()

	table.SetBorders(true)

	table.SetSelectable(true, false).
		SetSelectedFunc(func(row, col int) {
			if row == 0 {
				return
			}
			itemID := table.GetCell(row, 0).Text
			showItemDataDialog(app, userUID, token, itemID, table, message)
		}).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyTab {
				app.SetFocus(form)
			}
		})

	if err := loadUserData(table, userUID, token); err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error loading data: %v", err))
	}

	form.
		AddButton("Add Text", func() {
			showAddTextDialog(app, userUID, token, message, table)
		}).
		AddButton("Add File", func() {
			showAddFileDialog(app, userUID, token, message, table)
		}).
		AddButton("Exit", func() {
			app.Stop()
		})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyBacktab {
			app.SetFocus(table)
			return nil
		}
		return event
	})

	messageHint := tview.NewTextView().
		SetText("Use ↑/↓ for change select item in table. [Tab] to switch on menu. [Shift+Tab] to switch on table]").
		SetTextAlign(tview.AlignCenter)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 2, true).
		AddItem(form, 7, 1, false).
		AddItem(message, 1, 1, false).
		AddItem(messageHint, 1, 1, false)

	flex.SetBorder(true).
		SetTitle(" Your data ").
		SetTitleAlign(tview.AlignCenter)

	pages.AddPage("data_screen", flex, true, true)
	pages.SwitchToPage("data_screen")

	app.SetFocus(table)
}

// showAddTextDialog модальное окно для сохранения текста
func showAddTextDialog(app *tview.Application, userUID, token string, message *tview.TextView, table *tview.Table) {
	inputNameField := tview.NewInputField().
		SetLabel("Enter name: ").
		SetFieldWidth(40)

	inputField := tview.NewInputField().
		SetLabel("Enter text: ").
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputNameField).
		AddFormItem(inputField).
		AddButton("Save", func() {
			text := inputField.GetText()
			name := inputNameField.GetText()

			err := saveData(userUID, token, "text", name, []byte(text))
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
			} else {
				message.SetTextColor(tcell.ColorGreen).SetText("Text saved!")
				_ = loadUserData(table, userUID, token)
			}
			closeDialog("dialog_add_text")
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_add_text")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Add new text ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_add_text", dialogFlex, true, true)
	pages.SwitchToPage("dialog_add_text")
	app.SetFocus(dialogForm)
}

// showAddFileDialog модальное окно для сохранения файла
func showAddFileDialog(app *tview.Application, userUID, token string, message *tview.TextView, table *tview.Table) {
	inputNameField := tview.NewInputField().
		SetLabel("Enter name: ").
		SetFieldWidth(40)

	filePathField := tview.NewInputField().
		SetLabel("File path: ").
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputNameField).
		AddFormItem(filePathField).
		AddButton("Save", func() {
			filePath := filePathField.GetText()
			name := inputNameField.GetText()

			data, err := os.ReadFile(filePath)
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Failed to read file: %v", err))
				closeDialog("dialog_add_file")
				return
			}
			err = saveData(userUID, token, "file", name, data)
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
			} else {
				message.SetTextColor(tcell.ColorGreen).SetText("File saved!")
				_ = loadUserData(table, userUID, token)
			}
			closeDialog("dialog_add_file")
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_add_file")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Add new file ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_add_file", dialogFlex, true, true)
	pages.SwitchToPage("dialog_add_file")
	app.SetFocus(dialogForm)
}

// showItemDataDialog получение item для отображения или скачивания
func showItemDataDialog(app *tview.Application, userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := dataClient.GetData(ctx, &pb.GetDataRequest{Id: itemID})
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error getting data: %v", err))
		return
	}

	switch resp.Type {
	case "text":
		textData := resp.GetTextData()
		showTextContentModal(app, userUID, token, itemID, textData, table, message)
	case "file":
		fileData := resp.GetFileData()
		showFileContentModal(app, userUID, token, itemID, fileData, table, message)
	default:
		message.SetTextColor(tcell.ColorYellow).SetText(fmt.Sprintf("Unknown data type: %s", resp.Type))
	}
}

// showTextContentModal модальное окно для отображения текста
func showTextContentModal(app *tview.Application, userUID, token, itemID string, textData string, table *tview.Table, message *tview.TextView) {
	textView := tview.NewTextView().
		SetText(textData).
		SetWrap(true).
		SetScrollable(true)

	textView.SetBorder(true).
		SetTitle(" Text content ").
		SetTitleAlign(tview.AlignCenter)

	form := tview.NewForm().
		AddButton("Delete", func() {
			if err := deleteData(userUID, token, itemID); err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Delete error: %v", err))
			} else {
				message.SetTextColor(tcell.ColorGreen).SetText("Delete success!")
			}
			_ = loadUserData(table, userUID, token)
			closeDialog("dialog_view_text")
		}).
		AddButton("Close", func() {
			closeDialog("dialog_view_text")
		})

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, false).
		AddItem(form, 3, 1, true)

	pages.AddPage("dialog_view_text", dialogFlex, true, true)
	pages.SwitchToPage("dialog_view_text")
	app.SetFocus(form)
}

// showFileContentModal модальное окно для скачивания файла
func showFileContentModal(app *tview.Application, userUID, token, itemID string, fileData []byte, table *tview.Table, message *tview.TextView) {
	filePathField := tview.NewInputField().
		SetLabel("Save to file path: ").
		SetFieldWidth(40)

	form := tview.NewForm().
		AddFormItem(filePathField).
		AddButton("Save", func() {
			path := filePathField.GetText()
			err := os.WriteFile(path, fileData, 0644)
			if err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error saving file: %v", err))
			} else {
				message.SetTextColor(tcell.ColorGreen).SetText(fmt.Sprintf("File saved to: %s", path))
			}
			closeDialog("dialog_view_file")
		}).
		AddButton("Delete", func() {
			if err := deleteData(userUID, token, itemID); err != nil {
				message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Delete error: %v", err))
			} else {
				message.SetTextColor(tcell.ColorGreen).SetText("Delete success!")
			}
			_ = loadUserData(table, userUID, token)
			closeDialog("dialog_view_text")
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_view_file")
		})

	form.SetBorder(true).
		SetTitle(" File content ").
		SetTitleAlign(tview.AlignCenter)

	messageFileSize := tview.NewTextView().
		SetText(fmt.Sprintf("File size: %d bytes", len(fileData))).
		SetTextAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(messageFileSize, 1, 1, false)

	pages.AddPage("dialog_view_file", dialogFlex, true, true)
	pages.SwitchToPage("dialog_view_file")
	app.SetFocus(form)
}

// loadUserData загрузка данных пользователя для таблицы
func loadUserData(table *tview.Table, userUID, token string) error {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	resp, err := dataClient.GetUserDataList(
		ctx,
		&pb.GetUserDataListRequest{},
	)
	if err != nil {
		return err
	}

	table.Clear()

	table.SetCell(0, 0, tview.NewTableCell("ID").SetSelectable(false)).
		SetCell(0, 1, tview.NewTableCell("TYPE").SetSelectable(false)).
		SetCell(0, 2, tview.NewTableCell("NAME").SetSelectable(false))

	for i, item := range resp.Items {
		table.SetCell(i+1, 0, tview.NewTableCell(item.Id))
		table.SetCell(i+1, 1, tview.NewTableCell(item.Type))
		table.SetCell(i+1, 2, tview.NewTableCell(item.Name))
	}
	return nil
}

// saveData делает запрос на сохранение данных
func saveData(userUID, token, dataType, name string, data []byte) error {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := dataClient.SaveData(ctx, &pb.SaveDataRequest{
		UserUid: userUID,
		Type:    dataType,
		Name:    name,
		Data:    data,
	})
	return err
}

// deleteData делает запрос на удаление данных
func deleteData(userUID, token, itemID string) error {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err := dataClient.DeleteData(ctx, &pb.DeleteDataRequest{Id: itemID})
	return err
}

// closeDialog закрывает модальную страницу и возвращает на экран data_screen
func closeDialog(pageName string) {
	pages.RemovePage(pageName)
	pages.SwitchToPage("data_screen")
}
