package discord

import (
	"context"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v2/api/webhook"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/model"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

// Notifier .
type Notifier struct {
	conf   *config.DiscordConfig
	tmpl   *template.Template
	logger log.Logger
	client *webhook.Client
}

// New .
func New(conf *config.DiscordConfig, t *template.Template, l log.Logger) (*Notifier, error) {
	return &Notifier{
		conf:   conf,
		tmpl:   t,
		logger: l,
		client: webhook.New(conf.WebhookID, conf.Token),
	}, nil
}

func truncateAlerts(maxAlerts uint64, alerts []*types.Alert) ([]*types.Alert, uint64) {
	if maxAlerts != 0 && uint64(len(alerts)) > maxAlerts {
		return alerts[:maxAlerts], uint64(len(alerts)) - maxAlerts
	}
	return alerts, 0
}

// Notify implements the Notifier interface.
func (n *Notifier) Notify(_ context.Context, alerts ...*types.Alert) (bool, error) {
	maxAlerts := 9
	if len(alerts) == 10 {
		maxAlerts = 10
	}
	truncatedAlerts, numTruncated := truncateAlerts(uint64(maxAlerts), alerts)
	data := webhook.ExecuteData{}
	for _, alert := range truncatedAlerts {
		embed := discord.NewEmbed()
		embed.Title = alert.Name()
		embed.Timestamp = discord.NowTimestamp()

		if v, ok := alert.Labels[model.LabelName("node")]; ok {
			embed.Footer = &discord.EmbedFooter{
				Text: string(v),
			}
		}

		if v, ok := alert.Annotations[model.LabelName(alert.Status())]; ok {
			embed.Description = string(v)
		}

		switch alert.Status() {
		case model.AlertFiring:
			embed.Color = 0xE84757
		case model.AlertResolved:
			embed.Color = 0x3DAF6D
		}

		var labels strings.Builder
		for k, v := range alert.Labels {
			labels.WriteString(string(k))
			labels.Write([]byte(": "))
			labels.WriteString(string(v))
			labels.Write([]byte("\n"))
		}

		/*var annotations strings.Builder
		for k, v := range alert.Annotations {
			annotations.WriteString(string(k))
			annotations.Write([]byte(": "))
			annotations.WriteString(string(v))
			annotations.Write([]byte("\n"))
		}*/

		embed.Fields = []discord.EmbedField{
			{
				Name:  "**\u00BB Labels**",
				Value: strings.TrimRight(labels.String(), "\n"),
			},
			/*{
				Name:  "**\u00BB Annotations**",
				Value: strings.TrimRight(annotations.String(), "\n"),
			},*/
		}
		data.Embeds = append(data.Embeds, *embed)
	}
	if numTruncated > 0 {
		embed := discord.NewEmbed()
		embed.Description = "and " + strconv.Itoa(int(numTruncated)) + " more..."
		data.Embeds = append(data.Embeds, *embed)
	}

	if err := n.client.Execute(data); err != nil {
		return false, err
	}
	return true, nil
}
