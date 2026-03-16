package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Dawgomatic/Xagent/pkg/bus"
	"github.com/Dawgomatic/Xagent/pkg/config"
	"github.com/Dawgomatic/Xagent/pkg/logger"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"google.golang.org/protobuf/proto"
)

type WhatsAppChannel struct {
	*BaseChannel
	client *whatsmeow.Client
	config config.WhatsAppConfig
	mu     sync.Mutex
}

func NewWhatsAppChannel(cfg config.WhatsAppConfig, bus *bus.MessageBus) (*WhatsAppChannel, error) {
	base := NewBaseChannel("whatsapp", cfg, bus, cfg.AllowFrom)

	return &WhatsAppChannel{
		BaseChannel: base,
		config:      cfg,
	}, nil
}

func (c *WhatsAppChannel) Start(ctx context.Context) error {
	logger.InfoCF("whatsapp", "Starting native WhatsApp channel...", nil)

	// Set up SQLite Database for session storage
	dbPath := c.config.SessionDB
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".xagent", "whatsapp.db")
		os.MkdirAll(filepath.Dir(dbPath), 0755)
	}

	dbLog := waLog.Stdout("Database", "WARN", true)
	// SWE100821: Updated to current whatsmeow API — context required
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
	if err != nil {
		return fmt.Errorf("failed to init whatsapp database: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get whatsapp device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "WARN", true)
	c.client = whatsmeow.NewClient(deviceStore, clientLog)
	c.client.AddEventHandler(c.eventHandler)

	if c.client.Store.ID == nil {
		// New device, need to pair via QR code
		qrChan, _ := c.client.GetQRChannel(context.Background())
		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect whatsapp client: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				logger.InfoCF("whatsapp", "\n>>> WHATSAPP QR CODE SCANNED REQUIRED <<<\nScan the following string as a QR code or use a compatible terminal.", nil)
				fmt.Println("================================================================")
				fmt.Println("Scan this QR Code in WhatsApp -> Linked Devices -> Link a Device")
				fmt.Println(evt.Code)
				fmt.Println("================================================================")
			} else {
				logger.InfoCF("whatsapp", "Login event", map[string]interface{}{"event": evt.Event})
			}
		}
	} else {
		// Already logged in
		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect whatsapp client: %w", err)
		}
		logger.InfoCF("whatsapp", "WhatsApp channel connected natively.", nil)
	}

	c.setRunning(true)

	// Keep the channel alive until context is cancelled
	go func() {
		<-ctx.Done()
		c.Stop(context.Background())
	}()

	return nil
}

func (c *WhatsAppChannel) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setRunning(false)
	if c.client != nil {
		c.client.Disconnect()
		logger.InfoCF("whatsapp", "WhatsApp channel disconnected.", nil)
	}

	return nil
}

func (c *WhatsAppChannel) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		c.handleMessageEvent(v)
	}
}

func (c *WhatsAppChannel) handleMessageEvent(msg *events.Message) {
	// Ignore messages sent by us
	if msg.Info.IsFromMe {
		return
	}

	senderID := msg.Info.Sender.ToNonAD().String()
	chatID := msg.Info.Chat.ToNonAD().String()

	// Extract text content
	var content string
	if msg.Message.Conversation != nil {
		content = msg.Message.GetConversation()
	} else if msg.Message.ExtendedTextMessage != nil {
		content = msg.Message.ExtendedTextMessage.GetText()
	} else if msg.Message.ImageMessage != nil {
		content = msg.Message.ImageMessage.GetCaption()
	} else {
		// Unsupported message type
		return
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	metadata := map[string]string{
		"message_id": msg.Info.ID,
		"push_name":  msg.Info.PushName,
	}

	logger.DebugCF("whatsapp", "Received message", map[string]interface{}{
		"sender":  senderID,
		"chat":    chatID,
		"content": content[:min(len(content), 50)],
	})

	// Route to Xagent core via BaseChannel handler
	c.HandleMessage(senderID, chatID, content, nil, metadata)
}

func (c *WhatsAppChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || !c.client.IsConnected() {
		return fmt.Errorf("whatsapp client not connected")
	}

	// Parse the target chat JID
	jid, err := types.ParseJID(msg.ChatID)
	if err != nil {
		return fmt.Errorf("failed to parse chat JID %s: %w", msg.ChatID, err)
	}

	waMsg := &waProto.Message{
		Conversation: proto.String(msg.Content),
	}

	_, err = c.client.SendMessage(ctx, jid, waMsg)
	if err != nil {
		return fmt.Errorf("failed to send whatsapp message: %w", err)
	}

	logger.DebugCF("whatsapp", "Sent message", map[string]interface{}{
		"chat": msg.ChatID,
	})

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
