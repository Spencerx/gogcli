package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"google.golang.org/api/calendar/v3"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type CalendarCreateCmd struct {
	CalendarID            string   `arg:"" name:"calendarId" help:"Calendar ID"`
	Summary               string   `name:"summary" help:"Event summary/title"`
	From                  string   `name:"from" help:"Start time (RFC3339)"`
	To                    string   `name:"to" help:"End time (RFC3339)"`
	Description           string   `name:"description" help:"Description"`
	Location              string   `name:"location" help:"Location"`
	Attendees             string   `name:"attendees" help:"Comma-separated attendee emails"`
	AllDay                bool     `name:"all-day" help:"All-day event (use date-only in --from/--to)"`
	Recurrence            []string `name:"rrule" help:"Recurrence rules (e.g., 'RRULE:FREQ=MONTHLY;BYMONTHDAY=11'). Can be repeated."`
	Reminders             []string `name:"reminder" help:"Custom reminders as method:duration (e.g., popup:30m, email:1d). Can be repeated (max 5)."`
	ColorId               string   `name:"event-color" help:"Event color ID (1-11). Use 'gog calendar colors' to see available colors."`
	Visibility            string   `name:"visibility" help:"Event visibility: default, public, private, confidential"`
	Transparency          string   `name:"transparency" help:"Show as busy (opaque) or free (transparent). Aliases: busy, free"`
	SendUpdates           string   `name:"send-updates" help:"Notification mode: all, externalOnly, none (default: all)"`
	GuestsCanInviteOthers *bool    `name:"guests-can-invite" help:"Allow guests to invite others"`
	GuestsCanModify       *bool    `name:"guests-can-modify" help:"Allow guests to modify event"`
	GuestsCanSeeOthers    *bool    `name:"guests-can-see-others" help:"Allow guests to see other guests"`
	WithMeet              bool     `name:"with-meet" help:"Create a Google Meet video conference for this event"`
	SourceUrl             string   `name:"source-url" help:"URL where event was created/imported from"`
	SourceTitle           string   `name:"source-title" help:"Title of the source"`
	Attachments           []string `name:"attachment" help:"File attachment URL (can be repeated)"`
	PrivateProps          []string `name:"private-prop" help:"Private extended property (key=value, can be repeated)"`
	SharedProps           []string `name:"shared-prop" help:"Shared extended property (key=value, can be repeated)"`
}

func (c *CalendarCreateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	calendarID := strings.TrimSpace(c.CalendarID)
	if calendarID == "" {
		return usage("empty calendarId")
	}

	if strings.TrimSpace(c.Summary) == "" || strings.TrimSpace(c.From) == "" || strings.TrimSpace(c.To) == "" {
		return usage("required: --summary, --from, --to")
	}

	colorId, err := validateColorId(c.ColorId)
	if err != nil {
		return err
	}
	visibility, err := validateVisibility(c.Visibility)
	if err != nil {
		return err
	}
	transparency, err := validateTransparency(c.Transparency)
	if err != nil {
		return err
	}
	sendUpdates, err := validateSendUpdates(c.SendUpdates)
	if err != nil {
		return err
	}
	reminders, err := buildReminders(c.Reminders)
	if err != nil {
		return err
	}

	svc, err := newCalendarService(ctx, account)
	if err != nil {
		return err
	}

	event := &calendar.Event{
		Summary:            strings.TrimSpace(c.Summary),
		Description:        strings.TrimSpace(c.Description),
		Location:           strings.TrimSpace(c.Location),
		Start:              buildEventDateTime(c.From, c.AllDay),
		End:                buildEventDateTime(c.To, c.AllDay),
		Attendees:          buildAttendees(c.Attendees),
		Recurrence:         buildRecurrence(c.Recurrence),
		Reminders:          reminders,
		ColorId:            colorId,
		Visibility:         visibility,
		Transparency:       transparency,
		ConferenceData:     buildConferenceData(c.WithMeet),
		Attachments:        buildAttachments(c.Attachments),
		ExtendedProperties: buildExtendedProperties(c.PrivateProps, c.SharedProps),
	}
	if c.GuestsCanInviteOthers != nil {
		event.GuestsCanInviteOthers = c.GuestsCanInviteOthers
	}
	if c.GuestsCanModify != nil {
		event.GuestsCanModify = *c.GuestsCanModify
	}
	if c.GuestsCanSeeOthers != nil {
		event.GuestsCanSeeOtherGuests = c.GuestsCanSeeOthers
	}
	if strings.TrimSpace(c.SourceUrl) != "" {
		event.Source = &calendar.EventSource{
			Url:   strings.TrimSpace(c.SourceUrl),
			Title: strings.TrimSpace(c.SourceTitle),
		}
	}

	call := svc.Events.Insert(calendarID, event)
	if sendUpdates != "" {
		call = call.SendUpdates(sendUpdates)
	}
	if c.WithMeet {
		call = call.ConferenceDataVersion(1)
	}
	if len(event.Attachments) > 0 {
		call = call.SupportsAttachments(true)
	}
	created, err := call.Do()
	if err != nil {
		return err
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{"event": created})
	}
	printCalendarEvent(u, created)
	return nil
}

type CalendarUpdateCmd struct {
	CalendarID            string   `arg:"" name:"calendarId" help:"Calendar ID"`
	EventID               string   `arg:"" name:"eventId" help:"Event ID"`
	Summary               string   `name:"summary" help:"New summary/title (set empty to clear)"`
	From                  string   `name:"from" help:"New start time (RFC3339; set empty to clear)"`
	To                    string   `name:"to" help:"New end time (RFC3339; set empty to clear)"`
	Description           string   `name:"description" help:"New description (set empty to clear)"`
	Location              string   `name:"location" help:"New location (set empty to clear)"`
	Attendees             string   `name:"attendees" help:"Comma-separated attendee emails (replaces all; set empty to clear)"`
	AddAttendee           string   `name:"add-attendee" help:"Comma-separated attendee emails to add (preserves existing attendees)"`
	AllDay                bool     `name:"all-day" help:"All-day event (use date-only in --from/--to)"`
	Recurrence            []string `name:"rrule" help:"Recurrence rules (e.g., 'RRULE:FREQ=MONTHLY;BYMONTHDAY=11'). Can be repeated. Set empty to clear."`
	Reminders             []string `name:"reminder" help:"Custom reminders as method:duration (e.g., popup:30m, email:1d). Can be repeated (max 5). Set empty to clear."`
	ColorId               string   `name:"event-color" help:"Event color ID (1-11, or empty to clear)"`
	Visibility            string   `name:"visibility" help:"Event visibility: default, public, private, confidential"`
	Transparency          string   `name:"transparency" help:"Show as busy (opaque) or free (transparent). Aliases: busy, free"`
	GuestsCanInviteOthers *bool    `name:"guests-can-invite" help:"Allow guests to invite others"`
	GuestsCanModify       *bool    `name:"guests-can-modify" help:"Allow guests to modify event"`
	GuestsCanSeeOthers    *bool    `name:"guests-can-see-others" help:"Allow guests to see other guests"`
	Scope                 string   `name:"scope" help:"For recurring events: single, future, all" default:"all"`
	OriginalStartTime     string   `name:"original-start" help:"Original start time of instance (required for scope=single,future)"`
	PrivateProps          []string `name:"private-prop" help:"Private extended property (key=value, can be repeated)"`
	SharedProps           []string `name:"shared-prop" help:"Shared extended property (key=value, can be repeated)"`
}

func (c *CalendarUpdateCmd) Run(ctx context.Context, kctx *kong.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	calendarID := strings.TrimSpace(c.CalendarID)
	eventID := strings.TrimSpace(c.EventID)
	if calendarID == "" {
		return usage("empty calendarId")
	}
	if eventID == "" {
		return usage("empty eventId")
	}

	scope := strings.TrimSpace(strings.ToLower(c.Scope))
	if scope == "" {
		scope = scopeAll
	}
	switch scope {
	case scopeSingle:
		if strings.TrimSpace(c.OriginalStartTime) == "" {
			return usage("--original-start required when --scope=single")
		}
	case scopeFuture:
		if strings.TrimSpace(c.OriginalStartTime) == "" {
			return usage("--original-start required when --scope=future")
		}
	case scopeAll:
	default:
		return fmt.Errorf("invalid scope: %q (must be single, future, or all)", scope)
	}

	// If --all-day changed, require from/to to update both date/time fields.
	if flagProvided(kctx, "all-day") {
		if !flagProvided(kctx, "from") || !flagProvided(kctx, "to") {
			return usage("when changing --all-day, also provide --from and --to")
		}
	}

	// Cannot use both --attendees and --add-attendee at the same time.
	if flagProvided(kctx, "attendees") && flagProvided(kctx, "add-attendee") {
		return usage("cannot use both --attendees and --add-attendee; use --attendees to replace all, or --add-attendee to add")
	}

	patch, changed, err := c.buildUpdatePatch(kctx)
	if err != nil {
		return err
	}

	wantsAddAttendee := flagProvided(kctx, "add-attendee")
	if wantsAddAttendee && strings.TrimSpace(c.AddAttendee) == "" {
		return usage("empty --add-attendee")
	}

	if !changed && !wantsAddAttendee {
		return usage("no updates provided")
	}

	svc, err := newCalendarService(ctx, account)
	if err != nil {
		return err
	}

	// For --add-attendee, fetch current event to preserve existing attendees with metadata.
	if wantsAddAttendee {
		existing, getErr := svc.Events.Get(calendarID, eventID).Context(ctx).Do()
		if getErr != nil {
			return fmt.Errorf("failed to fetch current event: %w", getErr)
		}
		patch.Attendees = mergeAttendees(existing.Attendees, c.AddAttendee)
		changed = true
	}

	if !changed {
		return usage("no updates provided")
	}

	targetEventID, parentRecurrence, err := applyUpdateScope(ctx, svc, calendarID, eventID, scope, c.OriginalStartTime, patch)
	if err != nil {
		return err
	}

	updated, err := svc.Events.Patch(calendarID, targetEventID, patch).Do()
	if err != nil {
		return err
	}
	if scope == scopeFuture {
		if err := truncateParentRecurrence(ctx, svc, calendarID, eventID, parentRecurrence, c.OriginalStartTime); err != nil {
			return err
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{"event": updated})
	}
	printCalendarEvent(u, updated)
	return nil
}

func (c *CalendarUpdateCmd) buildUpdatePatch(kctx *kong.Context) (*calendar.Event, bool, error) {
	patch := &calendar.Event{}
	changed := false

	if flagProvided(kctx, "summary") {
		patch.Summary = strings.TrimSpace(c.Summary)
		changed = true
	}
	if flagProvided(kctx, "description") {
		patch.Description = strings.TrimSpace(c.Description)
		changed = true
	}
	if flagProvided(kctx, "location") {
		patch.Location = strings.TrimSpace(c.Location)
		changed = true
	}
	if flagProvided(kctx, "from") {
		patch.Start = buildEventDateTime(c.From, c.AllDay)
		changed = true
	}
	if flagProvided(kctx, "to") {
		patch.End = buildEventDateTime(c.To, c.AllDay)
		changed = true
	}
	if flagProvided(kctx, "attendees") {
		patch.Attendees = buildAttendees(c.Attendees)
		changed = true
	}
	if flagProvided(kctx, "rrule") {
		recurrence := buildRecurrence(c.Recurrence)
		if recurrence == nil {
			patch.Recurrence = []string{}
			patch.ForceSendFields = append(patch.ForceSendFields, "Recurrence")
		} else {
			patch.Recurrence = recurrence
		}
		changed = true
	}
	if flagProvided(kctx, "reminder") {
		reminders, err := buildReminders(c.Reminders)
		if err != nil {
			return nil, false, err
		}
		if reminders == nil {
			patch.Reminders = &calendar.EventReminders{UseDefault: true}
			patch.ForceSendFields = append(patch.ForceSendFields, "Reminders")
		} else {
			patch.Reminders = reminders
		}
		changed = true
	}
	if flagProvided(kctx, "event-color") {
		colorId, err := validateColorId(c.ColorId)
		if err != nil {
			return nil, false, err
		}
		patch.ColorId = colorId
		changed = true
	}
	if flagProvided(kctx, "visibility") {
		visibility, err := validateVisibility(c.Visibility)
		if err != nil {
			return nil, false, err
		}
		patch.Visibility = visibility
		changed = true
	}
	if flagProvided(kctx, "transparency") {
		transparency, err := validateTransparency(c.Transparency)
		if err != nil {
			return nil, false, err
		}
		patch.Transparency = transparency
		changed = true
	}
	if flagProvided(kctx, "guests-can-invite") {
		if c.GuestsCanInviteOthers != nil {
			patch.GuestsCanInviteOthers = c.GuestsCanInviteOthers
		}
		patch.ForceSendFields = append(patch.ForceSendFields, "GuestsCanInviteOthers")
		changed = true
	}
	if flagProvided(kctx, "guests-can-modify") {
		if c.GuestsCanModify != nil {
			patch.GuestsCanModify = *c.GuestsCanModify
		}
		patch.ForceSendFields = append(patch.ForceSendFields, "GuestsCanModify")
		changed = true
	}
	if flagProvided(kctx, "guests-can-see-others") {
		if c.GuestsCanSeeOthers != nil {
			patch.GuestsCanSeeOtherGuests = c.GuestsCanSeeOthers
		}
		patch.ForceSendFields = append(patch.ForceSendFields, "GuestsCanSeeOtherGuests")
		changed = true
	}
	if flagProvided(kctx, "private-prop") || flagProvided(kctx, "shared-prop") {
		patch.ExtendedProperties = buildExtendedProperties(c.PrivateProps, c.SharedProps)
		changed = true
	}

	return patch, changed, nil
}

func applyUpdateScope(ctx context.Context, svc *calendar.Service, calendarID, eventID, scope, originalStartTime string, patch *calendar.Event) (string, []string, error) {
	targetEventID := eventID
	var parentRecurrence []string

	if scope == scopeFuture {
		parent, err := svc.Events.Get(calendarID, eventID).Context(ctx).Do()
		if err != nil {
			return "", nil, err
		}
		if len(parent.Recurrence) == 0 {
			return "", nil, fmt.Errorf("event %s is not a recurring event", eventID)
		}
		parentRecurrence = parent.Recurrence
		recurrenceOverride := len(patch.Recurrence) > 0
		if !recurrenceOverride {
			for _, field := range patch.ForceSendFields {
				if field == "Recurrence" {
					recurrenceOverride = true
					break
				}
			}
		}
		if !recurrenceOverride {
			patch.Recurrence = parentRecurrence
		}
	}

	if scope == scopeSingle || scope == scopeFuture {
		instanceID, err := resolveRecurringInstanceID(ctx, svc, calendarID, eventID, originalStartTime)
		if err != nil {
			return "", nil, err
		}
		targetEventID = instanceID
	}

	return targetEventID, parentRecurrence, nil
}

func truncateParentRecurrence(ctx context.Context, svc *calendar.Service, calendarID, eventID string, parentRecurrence []string, originalStartTime string) error {
	truncated, err := truncateRecurrence(parentRecurrence, originalStartTime)
	if err != nil {
		return err
	}
	_, err = svc.Events.Patch(calendarID, eventID, &calendar.Event{Recurrence: truncated}).Context(ctx).Do()
	return err
}

type CalendarDeleteCmd struct {
	CalendarID        string `arg:"" name:"calendarId" help:"Calendar ID"`
	EventID           string `arg:"" name:"eventId" help:"Event ID"`
	Scope             string `name:"scope" help:"For recurring events: single, future, all" default:"all"`
	OriginalStartTime string `name:"original-start" help:"Original start time of instance (required for scope=single,future)"`
}

func (c *CalendarDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	calendarID := strings.TrimSpace(c.CalendarID)
	eventID := strings.TrimSpace(c.EventID)
	if calendarID == "" {
		return usage("empty calendarId")
	}
	if eventID == "" {
		return usage("empty eventId")
	}

	scope := strings.TrimSpace(strings.ToLower(c.Scope))
	if scope == "" {
		scope = scopeAll
	}
	switch scope {
	case scopeSingle:
		if strings.TrimSpace(c.OriginalStartTime) == "" {
			return usage("--original-start required when --scope=single")
		}
	case scopeFuture:
		if strings.TrimSpace(c.OriginalStartTime) == "" {
			return usage("--original-start required when --scope=future")
		}
	case scopeAll:
	default:
		return fmt.Errorf("invalid scope: %q (must be single, future, or all)", scope)
	}

	confirmMessage := fmt.Sprintf("delete event %s from calendar %s", eventID, calendarID)
	if scope == scopeSingle {
		confirmMessage = fmt.Sprintf("delete event %s (instance start %s) from calendar %s", eventID, c.OriginalStartTime, calendarID)
	}
	if scope == scopeFuture {
		confirmMessage = fmt.Sprintf("delete event %s (instance start %s) and all following from calendar %s", eventID, c.OriginalStartTime, calendarID)
	}
	if confirmErr := confirmDestructive(ctx, flags, confirmMessage); confirmErr != nil {
		return confirmErr
	}

	svc, err := newCalendarService(ctx, account)
	if err != nil {
		return err
	}

	targetEventID := eventID
	var parentRecurrence []string
	if scope == scopeFuture {
		parent, getErr := svc.Events.Get(calendarID, eventID).Context(ctx).Do()
		if getErr != nil {
			return getErr
		}
		if len(parent.Recurrence) == 0 {
			return fmt.Errorf("event %s is not a recurring event", eventID)
		}
		parentRecurrence = parent.Recurrence
	}
	if scope == scopeSingle || scope == scopeFuture {
		instanceID, resolveErr := resolveRecurringInstanceID(ctx, svc, calendarID, eventID, c.OriginalStartTime)
		if resolveErr != nil {
			return resolveErr
		}
		targetEventID = instanceID
	}

	if err := svc.Events.Delete(calendarID, targetEventID).Do(); err != nil {
		return err
	}
	if scope == scopeFuture {
		truncated, truncateErr := truncateRecurrence(parentRecurrence, c.OriginalStartTime)
		if truncateErr != nil {
			return truncateErr
		}
		_, patchErr := svc.Events.Patch(calendarID, eventID, &calendar.Event{Recurrence: truncated}).Context(ctx).Do()
		if patchErr != nil {
			return patchErr
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"deleted":    true,
			"calendarId": calendarID,
			"eventId":    targetEventID,
		})
	}
	u.Out().Printf("deleted\ttrue")
	u.Out().Printf("calendarId\t%s", calendarID)
	u.Out().Printf("eventId\t%s", targetEventID)
	return nil
}
