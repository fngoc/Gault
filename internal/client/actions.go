package client

import (
	pb "Gault/gen/go/api/proto/v1"
	"context"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"google.golang.org/grpc/metadata"
)

// registration запрос на регистрацию
func registration(app *tview.Application, login string, pass string, message *tview.TextView) {
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
	message.SetTextColor(tcell.ColorGreen).SetText("Registration successful!")
	showDataScreen(app, response.UserUid, response.Token, message)
}

// login запрос на авторизацию
func login(app *tview.Application, login, pass string, message *tview.TextView) {
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
	message.SetTextColor(tcell.ColorGreen).SetText("Login successful!")
	showDataScreen(app, response.UserUid, response.Token, message)
}

// saveText запрос на сохранение текста
func saveText(text, name, userUID, token string, table *tview.Table, message *tview.TextView) {
	err := saveData(userUID, token, "text", name, "", []byte(text))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Text saved!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_add_text")
}

// updateText запрос на обновление текста
func updateText(newText, userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	err := updateData(userUID, token, itemID, "text", "", []byte(newText))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Update error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Update success!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_edit_text")
	closeDialog("dialog_view_text")
}

// deleteText запрос на удаление текста
func deleteText(userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	if err := deleteData(userUID, token, itemID); err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Delete error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Delete success!")
	}
	_ = loadUserData(table, userUID, token)
	closeDialog("dialog_view_text")
}

// saveLoginAndPassword запрос на сохранение логина и пароля
func saveLoginAndPassword(text, name, userUID, token string, table *tview.Table, message *tview.TextView) {
	err := saveData(userUID, token, "password", name, "", []byte(text))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Password saved!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_add_text")
}

// updatePass запрос на обновление пароля
func updatePass(newText, userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	err := updateData(userUID, token, itemID, "password", "", []byte(newText))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Update error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Update success!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_edit_text")
	closeDialog("dialog_view_text")
}

// saveCard запрос на сохранение карты
func saveCard(text, name, userUID, token string, table *tview.Table, message *tview.TextView) {
	err := saveData(userUID, token, "card", name, "", []byte(text))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Password saved!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_add_text")
}

// updateCard запрос на обновление карты
func updateCard(newText, userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	err := updateData(userUID, token, itemID, "card", "", []byte(newText))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Update error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Update success!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_edit_text")
	closeDialog("dialog_view_text")
}

// saveFile запрос на сохранение файла
func saveFile(filePath, name, userUID, token string, table *tview.Table, message *tview.TextView) {
	err := saveData(userUID, token, "file", name, filePath, []byte(""))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Save error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("File saved!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_add_file")
}

// updateFile запрос на обновление файла
func updateFile(newPath, userUID, token, itemID string, table *tview.Table, message *tview.TextView) {
	err := updateData(userUID, token, itemID, "file", newPath, []byte(""))
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Replace error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("File replaced!")
		_ = loadUserData(table, userUID, token)
	}
	closeDialog("dialog_replace_file")
	closeDialog("dialog_view_file")
}

// downloadFile запрос на скачивание файла
func downloadFile(path string, fileData []byte, message *tview.TextView) {
	err := os.WriteFile(path, fileData, 0644)
	if err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Error saving file: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText(fmt.Sprintf("File saved to: %s", path))
	}
	closeDialog("dialog_view_file")
}

// deleteFile запрос на удаление файла
func deleteFile(userUID string, token string, itemID string, table *tview.Table, message *tview.TextView) {
	if err := deleteData(userUID, token, itemID); err != nil {
		message.SetTextColor(tcell.ColorRed).SetText(fmt.Sprintf("Delete error: %v", err))
	} else {
		message.SetTextColor(tcell.ColorGreen).SetText("Delete success!")
	}
	_ = loadUserData(table, userUID, token)
	closeDialog("dialog_view_text")
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
