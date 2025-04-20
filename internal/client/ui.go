package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/fngoc/gault/pkg/utils"

	pb "github.com/fngoc/gault/gen/go/api/proto/v1"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/grpc/metadata"
)

// Ключ шифрования
var aes string

// showLoginMenu экран логина/регистрации
func showLoginMenu(app *tview.Application, aesKey string) tview.Primitive {
	aes = aesKey

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
			login(app, loginField.GetText(), passField.GetText(), message)
		}).
		AddButton("Register", func() {
			registration(app, loginField.GetText(), passField.GetText(), message)
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
		AddButton("Add Login/Password", func() {
			showAddLoginPasswordDialog(app, userUID, token, message, table)
		}).
		AddButton("Add Card", func() {
			showAddCardDialog(app, userUID, token, message, table)
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
			saveText(inputField.GetText(), inputNameField.GetText(), userUID, token, table, message)
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

// showAddLoginPasswordDialog модальное окно для логина и пароля
func showAddLoginPasswordDialog(app *tview.Application, userUID, token string, message *tview.TextView, table *tview.Table) {
	inputLoginField := tview.NewInputField().
		SetLabel("Enter login: ").
		SetFieldWidth(40)

	inputPasswordField := tview.NewInputField().
		SetLabel("Enter password: ").
		SetMaskCharacter('*').
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputLoginField).
		AddFormItem(inputPasswordField).
		AddButton("Save", func() {
			saveLoginAndPassword(inputPasswordField.GetText(), inputLoginField.GetText(), userUID, token, table, message)
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

// showAddCardDialog модальное окно для добавления карт
func showAddCardDialog(app *tview.Application, userUID, token string, message *tview.TextView, table *tview.Table) {
	inputNameField := tview.NewInputField().
		SetLabel("Enter name card: ").
		SetFieldWidth(40)

	inputCardNumberField := tview.NewInputField().
		SetLabel("Enter card number: ").
		SetFieldWidth(40)

	inputDateNumberField := tview.NewInputField().
		SetLabel("Enter card date number: ").
		SetFieldWidth(40)

	inputCvcField := tview.NewInputField().
		SetLabel("Enter CVC number: ").
		SetMaskCharacter('*').
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputNameField).
		AddFormItem(inputCardNumberField).
		AddFormItem(inputDateNumberField).
		AddFormItem(inputCvcField).
		AddButton("Save", func() {
			saveCard(
				fmt.Sprintf("Number: [%s];\nDate number: [%s];\nCVC number: [%s];",
					inputCardNumberField.GetText(), inputDateNumberField.GetText(), inputCvcField.GetText()),
				inputNameField.GetText(), userUID, token, table, message)
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
			saveFile(filePathField.GetText(), inputNameField.GetText(), userUID, token, table, message)
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
	case "password":
		passData := resp.GetTextData()
		showPasswordContentModal(app, userUID, token, itemID, passData, table, message)
	case "card":
		cardData := resp.GetTextData()
		showCardContentModal(app, userUID, token, itemID, cardData, table, message)
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
		AddButton("Edit", func() {
			showEditTextDialog(app, userUID, token, itemID, textData, table, message)
		}).
		AddButton("Delete", func() {
			deleteText(userUID, token, itemID, table, message)
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

// showPasswordContentModal модальное окно для логина и пароля
func showPasswordContentModal(app *tview.Application, userUID, token, itemID string, textData string, table *tview.Table, message *tview.TextView) {
	passData, err := utils.Decrypt(textData, aes)
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error decrypting password: %v", err))
	}
	textView := tview.NewTextView().
		SetText(fmt.Sprintf("Password: %s", passData)).
		SetWrap(true).
		SetScrollable(true)

	textView.SetBorder(true).
		SetTitle(" Text content ").
		SetTitleAlign(tview.AlignCenter)

	form := tview.NewForm().
		AddButton("Edit", func() {
			showEditPasswordDialog(app, userUID, token, itemID, passData, table, message)
		}).
		AddButton("Delete", func() {
			deleteText(userUID, token, itemID, table, message)
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

// showCardContentModal модальное окно для логина и пароля
func showCardContentModal(app *tview.Application, userUID, token, itemID string, textData string, table *tview.Table, message *tview.TextView) {
	cardData, err := utils.Decrypt(textData, aes)
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error decrypting card: %v", err))
	}
	textView := tview.NewTextView().
		SetText(cardData).
		SetWrap(true).
		SetScrollable(true)

	textView.SetBorder(true).
		SetTitle(" Text content ").
		SetTitleAlign(tview.AlignCenter)

	form := tview.NewForm().
		AddButton("Edit", func() {
			showEditCardDialog(app, userUID, token, itemID, strings.Replace(cardData, "\n", " ", len(cardData)), table, message)
		}).
		AddButton("Delete", func() {
			deleteText(userUID, token, itemID, table, message)
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

// showEditTextDialog модальное окно для редактирования текста
func showEditTextDialog(app *tview.Application, userUID, token, itemID string, oldText string, table *tview.Table, message *tview.TextView) {
	inputField := tview.NewInputField().
		SetLabel("Edit text: ").
		SetText(oldText).
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Save", func() {
			updateText(inputField.GetText(), userUID, token, itemID, table, message)
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_edit_text")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Edit text ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_edit_text", dialogFlex, true, true)
	pages.SwitchToPage("dialog_edit_text")
	app.SetFocus(dialogForm)
}

// showEditPasswordDialog модальное окно для редактирования пароля
func showEditPasswordDialog(app *tview.Application, userUID, token, itemID string, oldPass string, table *tview.Table, message *tview.TextView) {
	inputField := tview.NewInputField().
		SetLabel("Edit password: ").
		SetText(oldPass).
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Save", func() {
			updatePass(inputField.GetText(), userUID, token, itemID, table, message)
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_edit_text")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Edit password ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_edit_text", dialogFlex, true, true)
	pages.SwitchToPage("dialog_edit_text")
	app.SetFocus(dialogForm)
}

// showEditCardDialog модальное окно для редактирования карты
func showEditCardDialog(app *tview.Application, userUID, token, itemID string, dataCard string, table *tview.Table, message *tview.TextView) {
	inputField := tview.NewInputField().
		SetLabel("Edit card: ").
		SetText(dataCard).
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(inputField).
		AddButton("Save", func() {
			updateCard(inputField.GetText(), userUID, token, itemID, table, message)
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_edit_text")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Edit card ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_edit_text", dialogFlex, true, true)
	pages.SwitchToPage("dialog_edit_text")
	app.SetFocus(dialogForm)
}

// showFileContentModal модальное окно для скачивания файла
func showFileContentModal(app *tview.Application, userUID, token, itemID string, fileData []byte, table *tview.Table, message *tview.TextView) {
	filePathField := tview.NewInputField().
		SetLabel("Save to file path: ").
		SetFieldWidth(40)

	form := tview.NewForm().
		AddFormItem(filePathField).
		AddButton("Save", func() {
			downloadFile(filePathField.GetText(), fileData, message)
		}).
		AddButton("Replace", func() {
			showReplaceFileDialog(app, userUID, token, itemID, fileData, table, message)
		}).
		AddButton("Delete", func() {
			deleteFile(userUID, token, itemID, table, message)
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

// showReplaceFileDialog модальное окно для выбора нового файла
func showReplaceFileDialog(app *tview.Application, userUID, token, itemID string, oldFileData []byte, table *tview.Table, message *tview.TextView) {
	newFilePathField := tview.NewInputField().
		SetLabel("New file path: ").
		SetFieldWidth(40)

	dialogForm := tview.NewForm().
		AddFormItem(newFilePathField).
		AddButton("Save", func() {
			updateFile(newFilePathField.GetText(), userUID, token, itemID, table, message)
		}).
		AddButton("Cancel", func() {
			closeDialog("dialog_replace_file")
		})

	dialogForm.SetBorder(true).
		SetTitle(" Replace file ").
		SetTitleAlign(tview.AlignCenter)

	dialogFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialogForm, 0, 1, true)

	pages.AddPage("dialog_replace_file", dialogFlex, true, true)
	pages.SwitchToPage("dialog_replace_file")
	app.SetFocus(dialogForm)
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
func saveData(userUID, token, dataType, name, filePath string, data []byte) error {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	if dataType == "text" {
		return sendSaveTextToServer(ctx, userUID, dataType, name, data, false)
	} else if dataType == "password" {
		return sendSaveTextToServer(ctx, userUID, dataType, name, data, true)
	} else if dataType == "card" {
		return sendSaveTextToServer(ctx, userUID, dataType, name, data, true)
	}
	return sendSaveBigFileToServer(ctx, filePath, userUID, dataType, name)
}

// updateData – делает запрос на обновление данных
func updateData(userUID, token, itemID, dataType, newPath string, data []byte) error {
	md := metadata.Pairs(
		"userUID", userUID,
		"authorization", token,
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	if dataType == "text" {
		return sendUpdateTextToServer(ctx, userUID, dataType, itemID, data, false)
	} else if dataType == "password" {
		return sendUpdateTextToServer(ctx, userUID, dataType, itemID, data, true)
	} else if dataType == "card" {
		return sendUpdateTextToServer(ctx, userUID, dataType, itemID, data, true)
	}
	return sendUpdateBigFileToServer(ctx, userUID, dataType, itemID, newPath)
}

// closeDialog закрывает модальную страницу и возвращает на экран data_screen
func closeDialog(pageName string) {
	pages.RemovePage(pageName)
	pages.SwitchToPage("data_screen")
}
