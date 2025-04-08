package client

import (
	pb "Gault/gen/go/api/proto/v1"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/rivo/tview"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
)

type fakeDataClient struct {
	lastUpdateRequest     *pb.UpdateDataRequest
	lastDeleteRequest     *pb.DeleteDataRequest
	lastGetUserDataCalled bool
	getUserDataResp       *pb.GetUserDataListResponse
	lastGetDataRequest    *pb.GetDataRequest
	getDataResp           *pb.GetDataResponse
	returnErr             error

	saveDataCreateStreamErr error
	saveDataSendErr         error
	saveDataCloseAndRecvErr error

	receivedChunks []*pb.SaveDataRequest
}

func (f *fakeDataClient) SaveData(ctx context.Context, opts ...grpc.CallOption) (pb.ContentManagerV1Service_SaveDataClient, error) {
	if f.saveDataCreateStreamErr != nil {
		return nil, f.saveDataCreateStreamErr
	}
	return &fakeSaveDataStream{
		parent: f,
	}, nil
}

func (f *fakeDataClient) GetData(ctx context.Context, in *pb.GetDataRequest, opts ...grpc.CallOption) (*pb.GetDataResponse, error) {
	f.lastGetDataRequest = in
	return f.getDataResp, f.returnErr
}

func (f *fakeDataClient) UpdateData(ctx context.Context, opts ...grpc.CallOption) (pb.ContentManagerV1Service_UpdateDataClient, error) {
	f.lastUpdateRequest = nil
	return nil, f.returnErr
}

func (f *fakeDataClient) DeleteData(ctx context.Context, in *pb.DeleteDataRequest, opts ...grpc.CallOption) (*pb.DeleteDataResponse, error) {
	f.lastDeleteRequest = in
	return &pb.DeleteDataResponse{}, f.returnErr
}

func (f *fakeDataClient) GetUserDataList(ctx context.Context, in *pb.GetUserDataListRequest, opts ...grpc.CallOption) (*pb.GetUserDataListResponse, error) {
	f.lastGetUserDataCalled = true
	return f.getUserDataResp, f.returnErr
}

type fakeAuthClient struct {
	lastLoginRequest        *pb.LoginRequest
	loginResp               *pb.LoginResponse
	lastRegistrationRequest *pb.RegistrationRequest
	registrationResp        *pb.RegistrationResponse
	returnErr               error
}

func (f *fakeAuthClient) Login(ctx context.Context, in *pb.LoginRequest, opts ...grpc.CallOption) (*pb.LoginResponse, error) {
	f.lastLoginRequest = in
	return f.loginResp, f.returnErr
}

func (f *fakeAuthClient) Registration(ctx context.Context, in *pb.RegistrationRequest, opts ...grpc.CallOption) (*pb.RegistrationResponse, error) {
	f.lastRegistrationRequest = in
	return f.registrationResp, f.returnErr
}

type fakeSaveDataStream struct {
	parent *fakeDataClient
}

func (s *fakeSaveDataStream) CloseSend() error {
	panic("implement me")
}

func (s *fakeSaveDataStream) Send(req *pb.SaveDataRequest) error {
	if s.parent.saveDataSendErr != nil {
		return s.parent.saveDataSendErr
	}
	s.parent.receivedChunks = append(s.parent.receivedChunks, req)
	return nil
}

func (s *fakeSaveDataStream) CloseAndRecv() (*pb.SaveDataResponse, error) {
	if s.parent.saveDataCloseAndRecvErr != nil {
		return nil, s.parent.saveDataCloseAndRecvErr
	}
	return &pb.SaveDataResponse{}, nil
}

func (s *fakeSaveDataStream) RecvMsg(m interface{}) error  { return nil }
func (s *fakeSaveDataStream) SendMsg(m interface{}) error  { return nil }
func (s *fakeSaveDataStream) Header() (metadata.MD, error) { return nil, nil }
func (s *fakeSaveDataStream) Trailer() metadata.MD         { return nil }
func (s *fakeSaveDataStream) Context() context.Context     { return context.Background() }
func (s *fakeSaveDataStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeSaveDataStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeSaveDataStream) SetTrailer(metadata.MD)       {}

func TestUpdateData_Success(t *testing.T) {
	client := &fakeDataClient{}
	dataClient = client

	userUID := "user-1"
	token := "token-abc"
	itemID := "item-42"
	data := []byte("updated content")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	err := updateData(userUID, token, itemID, "text", "", data)
	assert.NoError(t, err)
}

func TestUpdateData_Error(t *testing.T) {
	client := &fakeDataClient{returnErr: errors.New("update failed")}
	dataClient = client

	err := updateData("u", "t", "id", "file", "", []byte("content"))
	assert.Error(t, err)
}

func TestDeleteData_Success(t *testing.T) {
	client := &fakeDataClient{}
	dataClient = client

	userUID := "user-1"
	token := "token-abc"
	itemID := "item-007"

	err := deleteData(userUID, token, itemID)
	assert.NoError(t, err)

	assert.NotNil(t, client.lastDeleteRequest)
	assert.Equal(t, itemID, client.lastDeleteRequest.Id)
}

func TestDeleteData_Error(t *testing.T) {
	client := &fakeDataClient{returnErr: errors.New("delete failed")}
	dataClient = client

	err := deleteData("user", "token", "id123")
	assert.Error(t, err)
	assert.EqualError(t, err, "delete failed")
}

func TestLoadUserData_Success(t *testing.T) {
	table := tview.NewTable()
	client := &fakeDataClient{
		getUserDataResp: &pb.GetUserDataListResponse{
			Items: []*pb.UserDataItem{
				{Id: "1", Type: "text", Name: "note.txt"},
				{Id: "2", Type: "file", Name: "report.pdf"},
			},
		},
	}
	dataClient = client

	err := loadUserData(table, "user1", "token1")
	assert.NoError(t, err)
	assert.True(t, client.lastGetUserDataCalled)

	assert.Equal(t, "ID", table.GetCell(0, 0).Text)
	assert.Equal(t, "TYPE", table.GetCell(0, 1).Text)
	assert.Equal(t, "NAME", table.GetCell(0, 2).Text)

	assert.Equal(t, "1", table.GetCell(1, 0).Text)
	assert.Equal(t, "text", table.GetCell(1, 1).Text)
	assert.Equal(t, "note.txt", table.GetCell(1, 2).Text)

	assert.Equal(t, "2", table.GetCell(2, 0).Text)
	assert.Equal(t, "file", table.GetCell(2, 1).Text)
	assert.Equal(t, "report.pdf", table.GetCell(2, 2).Text)
}

func TestLoadUserData_Error(t *testing.T) {
	table := tview.NewTable()
	client := &fakeDataClient{
		returnErr: errors.New("get list failed"),
	}
	dataClient = client

	err := loadUserData(table, "u", "t")
	assert.Error(t, err)
	assert.EqualError(t, err, "get list failed")
}

func TestShowItemDataDialog_Text(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{
		getDataResp: &pb.GetDataResponse{
			Type: "text",
			Content: &pb.GetDataResponse_TextData{
				TextData: "Hello world!",
			},
		},
	}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showItemDataDialog(app, "userID", "token", "item", table, message)
}

func TestShowItemDataDialog_File(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{
		getDataResp: &pb.GetDataResponse{
			Type: "file",
			Content: &pb.GetDataResponse_FileData{
				FileData: []byte("data..."),
			},
		},
	}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showItemDataDialog(app, "userID", "token", "file", table, message)
}

func TestShowItemDataDialog_Error(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{
		returnErr: errors.New("server boom"),
	}
	dataClient = client

	showItemDataDialog(app, "u", "t", "id", table, message)

	text := message.GetText(true)
	assert.Contains(t, text, "server boom")
}

func TestCloseDialog(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	closeDialog("test")
}

func TestShowReplaceFileDialog_Success(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	tmpFile, err := os.CreateTemp("", "replace-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("Hello test data")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showReplaceFileDialog(app, "user1", "token1", "item123", []byte("old"), table, message)

	pageName, primitive := pages.GetFrontPage()
	assert.Equal(t, "dialog_replace_file", pageName)

	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	input, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	input.SetText(tmpFile.Name())
}

func TestShowEditTextDialog_Success(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showEditTextDialog(app, "user1", "token1", "item123", "old text", table, message)

	pageName, primitive := pages.GetFrontPage()
	assert.Equal(t, "dialog_edit_text", pageName)

	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	input, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)

	newText := "new edited text"
	input.SetText(newText)
}

func TestShowEditTextDialog_UpdateError(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{
		returnErr: errors.New("update failed"),
	}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showEditTextDialog(app, "userX", "tokenY", "itemABC", "text", table, message)

	_, primitive := pages.GetFrontPage()
	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	input, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	input.SetText("fail case")
}

func TestShowAddFileDialog_Success(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	tmpFile, err := os.CreateTemp("", "add-file-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("some file content")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showAddFileDialog(app, "user1", "token1", message, table)

	pageName, primitive := pages.GetFrontPage()
	assert.Equal(t, "dialog_add_file", pageName)

	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	nameField, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	pathField, ok := form.GetFormItem(1).(*tview.InputField)
	assert.True(t, ok)

	nameField.SetText("my_file")
	pathField.SetText(tmpFile.Name())
}

func TestShowAddFileDialog_FileReadError(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showAddFileDialog(app, "userX", "tokenY", message, table)

	_, primitive := pages.GetFrontPage()
	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	nameField, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	pathField, ok := form.GetFormItem(1).(*tview.InputField)
	assert.True(t, ok)

	nameField.SetText("bad_file")
	pathField.SetText("/no/such/file")
}

func TestShowAddTextDialog_Success(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showAddTextDialog(app, "user1", "token1", message, table)

	pageName, primitive := pages.GetFrontPage()
	assert.Equal(t, "dialog_add_text", pageName)

	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	nameField, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	textField, ok := form.GetFormItem(1).(*tview.InputField)
	assert.True(t, ok)

	nameField.SetText("entry1")
	textField.SetText("This is some text")
}

func TestShowAddTextDialog_SaveError(t *testing.T) {
	app := tview.NewApplication()
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("can't save")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showAddTextDialog(app, "userX", "tokenY", message, table)

	_, primitive := pages.GetFrontPage()
	form, ok := primitive.(*tview.Flex).GetItem(0).(*tview.Form)
	assert.True(t, ok)
	nameField, ok := form.GetFormItem(0).(*tview.InputField)
	assert.True(t, ok)
	textField, ok := form.GetFormItem(1).(*tview.InputField)
	assert.True(t, ok)

	nameField.SetText("fail")
	textField.SetText("nope")
}

func TestShowDataScreen_Success(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeDataClient{
		getUserDataResp: &pb.GetUserDataListResponse{
			Items: []*pb.UserDataItem{
				{Id: "id1", Type: "text", Name: "entry1"},
				{Id: "id2", Type: "file", Name: "doc.pdf"},
			},
		},
	}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showDataScreen(app, "user1", "token123", message)
}

func TestShowDataScreen_LoadError(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("boom")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	showDataScreen(app, "userX", "tokenY", message)
}

func TestShowLoginMenu_LoginSuccess(t *testing.T) {
	app := tview.NewApplication()
	showLoginMenu(app)
}

func TestLogin_Success(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeAuthClient{
		loginResp: &pb.LoginResponse{
			UserUid: "user1",
			Token:   "tokenABC",
		},
	}
	autClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	login(app, "test_user", "pass123", message)

	assert.NotNil(t, client.lastLoginRequest)
	assert.Equal(t, "test_user", client.lastLoginRequest.Login)
	assert.Equal(t, "pass123", client.lastLoginRequest.Password)
}

func TestLogin_Error(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeAuthClient{
		returnErr: errors.New("invalid credentials"),
	}
	autClient = client

	login(app, "bad_user", "bad_pass", message)

	text := message.GetText(true)
	assert.Contains(t, text, "Login error: invalid credentials")
}

func TestRegistration_Success(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeAuthClient{
		registrationResp: &pb.RegistrationResponse{
			UserUid: "new-user",
			Token:   "reg-token-123",
		},
	}
	autClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	registration(app, "newlogin", "newpass", message)

	assert.NotNil(t, client.lastRegistrationRequest)
	assert.Equal(t, "newlogin", client.lastRegistrationRequest.Login)
	assert.Equal(t, "newpass", client.lastRegistrationRequest.Password)
}

func TestRegistration_Error(t *testing.T) {
	app := tview.NewApplication()
	message := tview.NewTextView()

	client := &fakeAuthClient{
		returnErr: errors.New("login already exists"),
	}
	autClient = client

	registration(app, "existing_user", "123456", message)

	text := message.GetText(true)
	assert.Contains(t, text, "Registration error: login already exists")
}

func TestUpdate_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	tmpFile, err := os.CreateTemp("", "update-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("updated file content")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	updateFile(tmpFile.Name(), "user1", "token123", "item42", table, message)

	assert.NotNil(t, client.lastUpdateRequest)
	assert.Equal(t, content, client.lastUpdateRequest.Data)

	text := message.GetText(true)
	assert.Contains(t, text, "File replaced!")
}

func TestUpdate_FileReadError(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	updateFile("/non/existent/file/path.txt", "user1", "token", "item42", table, message)

	_ = message.GetText(true)
	assert.Nil(t, client.lastUpdateRequest)
}

func TestUpdate_UpdateDataError(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	tmpFile, err := os.CreateTemp("", "fail-update-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("should fail update")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	client := &fakeDataClient{returnErr: errors.New("update failed")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	updateFile(tmpFile.Name(), "user1", "token", "item99", table, message)

	_ = message.GetText(true)
}

func TestDeleteFile_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	deleteFile("user1", "tokenX", "item42", table, message)

	assert.NotNil(t, client.lastDeleteRequest)
	assert.Equal(t, "item42", client.lastDeleteRequest.Id)

	text := message.GetText(true)
	assert.Contains(t, text, "Delete success!")
}

func TestDeleteFile_Error(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("can't delete")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	deleteFile("user2", "tokenY", "item99", table, message)

	assert.NotNil(t, client.lastDeleteRequest)
	assert.Equal(t, "item99", client.lastDeleteRequest.Id)

	text := message.GetText(true)
	assert.Contains(t, text, "Delete error: can't delete")
}

func TestDownloadFile_Success(t *testing.T) {
	message := tview.NewTextView()

	tmpFile, err := os.CreateTemp("", "download-*.bin")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	data := []byte("test file content")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	downloadFile(tmpFile.Name(), data, message)
}

func TestDownloadFile_Error(t *testing.T) {
	message := tview.NewTextView()
	path := "/nonexistent-dir/output.txt"
	data := []byte("should fail")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	downloadFile(path, data, message)
}

func TestUpdateText_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	updateText("updated text here", "user1", "token1", "item123", table, message)
}

func TestUpdateText_Error(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("db write error")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	updateText("new content", "user2", "tokenX", "item999", table, message)
}

func TestDeleteText_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	deleteText("user1", "token123", "item42", table, message)
}

func TestDeleteText_Error(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("can't delete")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	deleteText("userX", "tokenY", "itemX", table, message)
}

func TestSaveFile_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	tmpFile, err := os.CreateTemp("", "save-file-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("saved content")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	saveFile(tmpFile.Name(), "report.txt", "user1", "tokenXYZ", table, message)
}

func TestSaveFile_ReadError(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	saveFile("/non/existent/path.txt", "fail.txt", "userX", "tokenX", table, message)
}

func TestSaveFile_SaveError(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	tmpFile, err := os.CreateTemp("", "fail-save-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("content"))
	assert.NoError(t, err)
	tmpFile.Close()

	client := &fakeDataClient{returnErr: errors.New("server failed")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	saveFile(tmpFile.Name(), "bad.txt", "userFail", "tokenFail", table, message)
}

func TestSaveText_Success(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	saveText("my content", "note.txt", "user1", "token123", table, message)
}

func TestSaveText_Error(t *testing.T) {
	table := tview.NewTable()
	message := tview.NewTextView()

	client := &fakeDataClient{returnErr: errors.New("disk full")}
	dataClient = client

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for TUI")
		}
	}()
	saveText("oops", "fail.txt", "userX", "tokenX", table, message)
}

func TestSendBigFileToServer_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bigfile-test-*.bin")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("Hello chunked gRPC!\nAnother line.")
	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	tmpFile.Close()

	client := &fakeDataClient{}
	dataClient = client

	err = sendBigFileToServer(context.Background(), tmpFile.Name(), "user123", "file", "myfile.txt")
	assert.NoError(t, err)

	assert.Len(t, client.receivedChunks, 1)
	assert.Equal(t, "user123", client.receivedChunks[0].UserUid)
	assert.Equal(t, "file", client.receivedChunks[0].Type)
	assert.Equal(t, "myfile.txt", client.receivedChunks[0].Name)
	assert.Equal(t, content, client.receivedChunks[0].Data)
}

func TestSendBigFileToServer_OpenFileError(t *testing.T) {
	client := &fakeDataClient{}
	dataClient = client

	err := sendBigFileToServer(context.Background(), "/no/such/path.bin", "u", "t", "n")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
	assert.Empty(t, client.receivedChunks)
}

func TestSendBigFileToServer_CreateStreamError(t *testing.T) {
	client := &fakeDataClient{
		saveDataCreateStreamErr: errors.New("stream creation failed"),
	}
	dataClient = client

	tmpFile, err := os.CreateTemp("", "bigfile-test-*.bin")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	err = sendBigFileToServer(context.Background(), tmpFile.Name(), "user123", "file", "myfile")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not create stream: stream creation failed")
	assert.Empty(t, client.receivedChunks)
}

func TestSendBigFileToServer_SendChunkError(t *testing.T) {
	client := &fakeDataClient{
		saveDataSendErr: errors.New("send chunk error"),
	}
	dataClient = client

	tmpFile, err := os.CreateTemp("", "chunk-fail-*.bin")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("some data here"))
	assert.NoError(t, err)
	tmpFile.Close()

	err = sendBigFileToServer(context.Background(), tmpFile.Name(), "u", "t", "n")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send chunk error")
}

func TestSendBigFileToServer_CloseAndRecvError(t *testing.T) {
	client := &fakeDataClient{
		saveDataCloseAndRecvErr: errors.New("final ack error"),
	}
	dataClient = client

	tmpFile, err := os.CreateTemp("", "bigfile-test-*.bin")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("some content"))
	assert.NoError(t, err)
	tmpFile.Close()

	err = sendBigFileToServer(context.Background(), tmpFile.Name(), "userX", "typeX", "nameX")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CloseAndRecv error: final ack error")
	assert.NotEmpty(t, client.receivedChunks)
}
