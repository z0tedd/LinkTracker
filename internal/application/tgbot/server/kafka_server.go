package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/central-university-dev/go-z0tedd/internal/application/dtos"
	"github.com/central-university-dev/go-z0tedd/internal/application/tgbot/helpers"
	"github.com/central-university-dev/go-z0tedd/internal/config"
)

type KafkaBotServer struct {
	logger   *slog.Logger
	consumer sarama.ConsumerGroup
	cfg      *config.Config
	tgAPI    *tgbotapi.BotAPI
}

func NewKafkaBotServer(cfg *config.Config, consumer sarama.ConsumerGroup,
	logger *slog.Logger, tgAPI *tgbotapi.BotAPI,
) (KafkaBotServer, error) {
	return KafkaBotServer{logger: logger, consumer: consumer, cfg: cfg, tgAPI: tgAPI}, nil
}

func (s *KafkaBotServer) Start(ctx context.Context) error {
	s.PrintLogo()

	producer, err := sarama.NewAsyncProducer(strings.Split(s.cfg.KafkaAddresses, ","), sarama.NewConfig())
	if err != nil {
		return err
	}

	handler := NewGroupHandler(ctx, s.logger, s.tgAPI, producer)

	err = s.consumer.Consume(ctx, []string{s.cfg.KafkaTopic}, handler)
	if err != nil {
		return fmt.Errorf("starting kafka server: %w", err)
	}

	return nil
}

// logo like in echo framework.
func (s *KafkaBotServer) PrintLogo() {
	fmt.Println(
		`
**++*+++++======+==+++++++++=++++++=*%%@========#@@%
**********++++++++++++=+%%%%#+==--+===%@@*=====@%@*=
**%#*+:-+***+++*#+#*:@#%=@::#%+:..:=====%%*=@@@%@#-:
%%@@@*::%.--+-==-=--.:-:.=..::%@*-.....:-#=#@@@@**+
+#%%%%.%%=@@@%-==@@=..::..#@@@@@+.#*-::...%@%@@@@==+
+++++**#%@@@+%:%@@=*@%@%%@%%+*-:=###*=::::%@%@%@***#
+++++++==*++%@.#@*##@@@@#=--=-=**%@%#*==--*#%@%%###%
++=+*-#%@-@%#%:%=*+::.:.::=+*##*=#%#%%%%#%##%-+++**#
+%==#%%@@%#*@*=**##****++++=+==+#%%%%%%%%%%%%%%%###*
=%@@====%@#*%%%%%%%%%%@@@@@@@***%%@%%%%#%##%%%%+****
==+%%=*%@@#*=....:#*#%*=-++*###%%%%%##%%%%%%#%%#**%%
====#%#@@%**#*+++#%%%%%%%@@%%+.:---++**#*###%#%%%%%%
====-#@@@@=+###%%%%%%%%%%%@@%%#++++===:.*-==+*+*####
-----*@@@*#####%%%%%%%###%%%%%%%%=-:--#%%#%*--:..--*
_________                               .___.__.__          
\_   ___ \______  ____   ____  ____   __| _/|__|  |   ____  
/    \  \|_  __ \/  _ \_/ ___\/  _ \ / __ | |  |  |  /  _ \ 
\     \___|  | \(  <_> )  \__(  <_> ) /_/ | |  |  |_(  <_> )
 \______  /__|   \____/ \___  >____/\____ | |__|____/\____/ 
        \/                  \/           \/                 
		`)
}

type GroupHandler struct {
	logger   *slog.Logger
	tgAPI    *tgbotapi.BotAPI
	ctx      context.Context
	producer sarama.AsyncProducer
}

func NewGroupHandler(ctx context.Context, logger *slog.Logger, tgAPI *tgbotapi.BotAPI, producer sarama.AsyncProducer) *GroupHandler {
	return &GroupHandler{logger: logger, ctx: ctx, producer: producer, tgAPI: tgAPI}
}

// Setup is called when the consumer group session is being set up.
func (h *GroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	h.logger.Info("consumer group session setup complete")
	return nil
}

// Cleanup is called when the consumer group session is ending.
func (h *GroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	h.logger.Info("consumer group session cleanup complete")
	return nil
}

// ConsumeClaim is called for each message received from Kafka.
func (h *GroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var encodedMessage dtos.UpdateDTO

		h.logger.Info("message received", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)

		err := encodedMessage.Decode(message.Value)
		if err != nil {
			records := make([]sarama.RecordHeader, len(message.Headers))

			for _, record := range message.Headers {
				if record != nil {
					records = append(records, *record)
				}
			}
			// Doesn't check successes and errors channel, because there is no sense in processing data from DLQ
			h.producer.Input() <- &sarama.ProducerMessage{
				Topic:   "DLQ",
				Headers: records,
				Value:   sarama.ByteEncoder(message.Value),
			}

			continue
		}

		for _, userID := range encodedMessage.TgChatIDs {
			err := helpers.SendMessage(h.tgAPI, userID, fmt.Sprintf("New update from your subcribed link: %s", encodedMessage.URL))
			if err != nil {
				h.logger.Error("Sending message", slog.Any("error", err.Error()))
			}
		}
		// Mark the message as processed
		session.MarkMessage(message, "")
	}

	return nil
}
