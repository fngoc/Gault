package client

import (
	pb "Gault/gen/go/api/proto/v1"
	"Gault/pkg/utils"
	"context"
	"fmt"
	"io"
	"os"
)

// sendSaveTextToServer отправляет текст через SaveData
func sendSaveTextToServer(ctx context.Context, userUID, dataType, name string, dataText []byte, isEncrypted bool) error {
	// Инициируем стрим
	stream, err := dataClient.SaveData(ctx)
	if err != nil {
		return err
	}

	if isEncrypted {
		password, err := utils.Encrypt(string(dataText), aes)
		if err != nil {
			return err
		}
		dataText = []byte(password)
	}

	// Посылаем один чанк
	req := &pb.SaveDataRequest{
		UserUid:     userUID,
		Type:        dataType,
		Name:        name,
		Data:        dataText,
		ChunkNumber: 1,
		TotalChunks: 1,
	}
	if err = stream.Send(req); err != nil {
		return err
	}

	// Закрываем стрим и ждём ответа
	_, err = stream.CloseAndRecv()
	if err != nil {
		return err
	}
	return nil
}

// sendUpdateTextToServer отправляет текст через UpdateData
func sendUpdateTextToServer(ctx context.Context, userUID, dataType, itemID string, dataText []byte, isEncrypt bool) error {
	// Инициируем стрим
	stream, err := dataClient.UpdateData(ctx)
	if err != nil {
		return err
	}

	if isEncrypt {
		password, err := utils.Encrypt(string(dataText), aes)
		if err != nil {
			return err
		}
		dataText = []byte(password)
	}

	// Посылаем один чанк
	req := &pb.UpdateDataRequest{
		UserUid:     userUID,
		Type:        dataType,
		DataUid:     itemID,
		Data:        dataText,
		ChunkNumber: 1,
		TotalChunks: 1,
	}
	if err = stream.Send(req); err != nil {
		return err
	}

	// Закрываем стрим и ждём ответа
	_, err = stream.CloseAndRecv()
	if err != nil {
		return err
	}
	return nil
}

// sendSaveBigFileToServer читает большой файл и грузит его чанками через SaveData
func sendSaveBigFileToServer(ctx context.Context, filePath, userUID, dataType, dataName string) error {
	// Открываем локальный файл
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Создаём gRPC стрим
	stream, err := dataClient.SaveData(ctx)
	if err != nil {
		return fmt.Errorf("could not create stream: %w", err)
	}

	// В цикле читаем файл и отправляем чанки по 1 MB
	const chunkSize = 1024 * 1024
	buf := make([]byte, chunkSize)

	for {
		n, readErr := f.Read(buf)
		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read file error: %w", readErr)
		}
		if n == 0 {
			// достигли конца файла
			break
		}

		req := &pb.SaveDataRequest{
			UserUid: userUID,
			Type:    dataType,
			Name:    dataName,
			Data:    buf[:n],
		}
		// Отправляем чанк в стрим
		if errSend := stream.Send(req); errSend != nil {
			return fmt.Errorf("send chunk error: %w", errSend)
		}

		if readErr == io.EOF {
			break
		}
	}

	// Закрываем стрим
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("CloseAndRecv error: %w", err)
	}
	fmt.Println("File uploaded successfully. SaveDataResponse:", resp)
	return nil
}

// sendUpdateBigFileToServer читает большой файл и грузит его чанками через UpdateData
func sendUpdateBigFileToServer(ctx context.Context, userUID, dataType, itemID, filePath string) error {
	// Открываем локальный файл
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Создаём gRPC стрим
	stream, err := dataClient.UpdateData(ctx)
	if err != nil {
		return fmt.Errorf("could not create stream: %w", err)
	}

	// В цикле читаем файл и отправляем чанки по 1 MB
	const chunkSize = 1024 * 1024
	buf := make([]byte, chunkSize)

	for {
		n, readErr := f.Read(buf)
		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read file error: %w", readErr)
		}
		if n == 0 {
			// достигли конца файла
			break
		}

		req := &pb.UpdateDataRequest{
			UserUid: userUID,
			Type:    dataType,
			DataUid: itemID,
			Data:    buf[:n],
		}
		// Отправляем чанк в стрим
		if errSend := stream.Send(req); errSend != nil {
			return fmt.Errorf("send chunk error: %w", errSend)
		}

		if readErr == io.EOF {
			break
		}
	}

	// Закрываем стрим
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("CloseAndRecv error: %w", err)
	}
	fmt.Println("File uploaded successfully. UpdateDataResponse:", resp)
	return nil
}
