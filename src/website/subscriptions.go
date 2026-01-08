package website

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/stripe/stripe-go/v84"
)

type PaymentHistoryItem struct {
	Date     string
	Amount   string
	CardInfo string
}

type ManageSubscriptionTemplateData struct {
	templates.BaseData
	SubscribeUrl          string
	CancelSubscriptionUrl string
	ResumeSubscriptionUrl string
	CurrentCurrencySymbol string
	CurrentAmount         string
	PaymentHistory        []PaymentHistoryItem
	CurrentPeriodEnd      string
	LastPaymentAmount     string
	LastPaymentMethod     string
}

func SubscriptionManage(c *RequestContext) ResponseData {

	// If the user just completed checkout, Stripe redirects with a session_id.
	// Verify it so we can show the correct "subscribed" view even if webhooks
	// haven't updated the DB yet.
	if c.CurrentUser != nil && !c.CurrentUser.IsSubscribed {
		if sessionID := c.Req.URL.Query().Get("session_id"); sessionID != "" {
			sc := stripe.NewClient(config.Config.Stripe.SecretKey)
			session, err := sc.V1CheckoutSessions.Retrieve(c, sessionID, nil)
			if err == nil && session.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
				c.CurrentUser.IsSubscribed = true
			}
		}
	}

	var history []PaymentHistoryItem
	currentCurrencySymbol := "$"
	currentAmount := "5.00"

	if c.CurrentUser != nil && c.CurrentUser.IsSubscribed {
		payments, _ := db.Query[models.UserPayment](c, c.Conn, "SELECT $columns FROM user_payment WHERE user_id = $1 ORDER BY paid_at DESC", c.CurrentUser.ID)
		if len(payments) > 0 {
			if strings.EqualFold(payments[0].Currency, "eur") {
				currentCurrencySymbol = "€"
			}
			currentAmount = fmt.Sprintf("%.2f", float64(payments[0].AmountCents)/100.0)
		}

		for _, p := range payments {
			sym := "$"
			if strings.EqualFold(p.Currency, "eur") {
				sym = "€"
			}
			card := ""
			if p.CardBrand != nil {
				card = strings.ToUpper(*p.CardBrand)
			}
			if p.CardLast4 != nil {
				if card != "" {
					card += " "
				}
				card += "•••• " + *p.CardLast4
			}
			history = append(history, PaymentHistoryItem{
				Date:     p.PaidAt.UTC().Format("Jan 2, 2006"),
				Amount:   fmt.Sprintf("%s%.2f", sym, float64(p.AmountCents)/100.0),
				CardInfo: card,
			})
		}
	}

	currentPeriodEnd := ""
	if c.CurrentUser != nil && c.CurrentUser.CurrentPeriodEnd != nil {
		currentPeriodEnd = c.CurrentUser.CurrentPeriodEnd.UTC().Format("Jan 2, 2006")
	}

	lastAmount := ""
	lastMethod := ""
	if len(history) > 0 {
		lastAmount = history[0].Amount
		lastMethod = history[0].CardInfo
	}

	var res ResponseData
	res.MustWriteTemplate("manage_subscription.html", ManageSubscriptionTemplateData{
		BaseData:              getBaseData(c, "Manage Membership", nil),
		SubscribeUrl:          hmnurl.BuildSubscriptionSubscribe(),
		CancelSubscriptionUrl: hmnurl.BuildSubscriptionCancel(),
		ResumeSubscriptionUrl: hmnurl.BuildSubscriptionResume(),
		CurrentCurrencySymbol: currentCurrencySymbol,
		CurrentAmount:         currentAmount,
		PaymentHistory:        history,
		CurrentPeriodEnd:      currentPeriodEnd,
		LastPaymentAmount:     lastAmount,
		LastPaymentMethod:     lastMethod,
	}, c.Perf)
	return res
}

func SubscriptionSubscribe(c *RequestContext) ResponseData {
	if c.CurrentUser.IsSubscribed {
		return c.Redirect(hmnurl.BuildSubscriptionManage(), http.StatusSeeOther)
	}

	sc := stripe.NewClient(config.Config.Stripe.SecretKey)

	priceID := c.Req.FormValue("price_id")
	if priceID == "" {
		priceID = config.Config.Stripe.PriceID
	}

	params := &stripe.CheckoutSessionCreateParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(hmnurl.BuildSubscriptionManage() + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String(hmnurl.BuildSubscriptionManage()),
		ClientReferenceID: stripe.String(strconv.Itoa(c.CurrentUser.ID)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		CustomerEmail: stripe.String(c.CurrentUser.Email),
		SubscriptionData: &stripe.CheckoutSessionCreateSubscriptionDataParams{
			Metadata: map[string]string{"user_id": strconv.Itoa(c.CurrentUser.ID)},
		},
	}

	p, err := sc.V1Prices.Retrieve(c, priceID, nil)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get requested price"))
	}
	targetCurrency := p.Currency
	targetAmount := p.UnitAmount

	if c.CurrentUser.StripeCustomerID != nil {
		params.Customer = stripe.String(*c.CurrentUser.StripeCustomerID)
		params.CustomerEmail = nil

		listParams := &stripe.CheckoutSessionListParams{
			Customer: stripe.String(*c.CurrentUser.StripeCustomerID),
			Status:   stripe.String(string(stripe.CheckoutSessionStatusOpen)),
		}

		iter := sc.V1CheckoutSessions.List(c, listParams)
		var existingURL string
		var outdatedSessionID string
		var listErr error
		iter(func(session *stripe.CheckoutSession, err error) bool {
			listErr = err
			if err == nil && session != nil {
				if session.Currency == targetCurrency && session.AmountTotal == targetAmount {
					existingURL = session.URL
				} else {
					outdatedSessionID = session.ID
				}
			}
			return false // pull only the first item
		})

		if listErr != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(listErr, "failed to list checkout sessions"))
		}

		if existingURL != "" {
			return c.Redirect(existingURL, http.StatusSeeOther)
		} else if outdatedSessionID != "" {
			_, _ = sc.V1CheckoutSessions.Expire(c, outdatedSessionID, nil)
		}
	}

	s, err := sc.V1CheckoutSessions.Create(c, params)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create checkout session"))
	}

	return c.Redirect(s.URL, http.StatusSeeOther)
}

func SubscriptionCancel(c *RequestContext) ResponseData {
	if c.CurrentUser.StripeSubscriptionID == nil {
		return c.Redirect(hmnurl.BuildSubscriptionManage(), http.StatusSeeOther)
	}

	sc := stripe.NewClient(config.Config.Stripe.SecretKey)

	params := &stripe.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	_, err := sc.V1Subscriptions.Update(c, *c.CurrentUser.StripeSubscriptionID, params)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to cancel subscription"))
	}

	_, err = c.Conn.Exec(c, "UPDATE hmn_user SET cancel_at_period_end = true WHERE id = $1", c.CurrentUser.ID)
	if err != nil {
		logging.Error().Err(err).Msg("failed to update user cancel_at_period_end optimistically")
	}

	return c.Redirect(hmnurl.BuildSubscriptionManage(), http.StatusSeeOther)
}

func SubscriptionResume(c *RequestContext) ResponseData {
	if c.CurrentUser.StripeSubscriptionID == nil {
		return c.Redirect(hmnurl.BuildSubscriptionManage(), http.StatusSeeOther)
	}

	if c.CurrentUser.CurrentPeriodEnd == nil || c.CurrentUser.CurrentPeriodEnd.Before(time.Now()) {
		return c.Redirect(hmnurl.BuildSubscriptionSubscribe(), http.StatusSeeOther)
	}

	sc := stripe.NewClient(config.Config.Stripe.SecretKey)

	params := &stripe.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	_, err := sc.V1Subscriptions.Update(c, *c.CurrentUser.StripeSubscriptionID, params)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to resume subscription"))
	}

	_, err = c.Conn.Exec(c, "UPDATE hmn_user SET cancel_at_period_end = false WHERE id = $1", c.CurrentUser.ID)
	if err != nil {
		logging.Error().Err(err).Msg("failed to update user cancel_at_period_end optimistically")
	}

	return c.Redirect(hmnurl.BuildSubscriptionManage(), http.StatusSeeOther)
}



func handleSubscriptionCreated(c *RequestContext, sc *stripe.Client, sub *stripe.Subscription) {
	// Link user if possible. Early registration.
	if uidStr, ok := sub.Metadata["user_id"]; ok {
		if uid, err := strconv.Atoi(uidStr); err == nil {
			renewalDate := getSubscriptionPeriodEnd(sub)
			_, err = c.Conn.Exec(c, `
				UPDATE hmn_user 
				SET 
					stripe_customer_id = $1, 
					stripe_subscription_id = $2, 
					subscription_status = $3,
					current_period_end = $4
				WHERE id = $5
			`, sub.Customer.ID, sub.ID, sub.Status, renewalDate, uid)
			if err != nil {
				logging.Error().Err(err).Int("userID", uid).Msg("failed to handle subscription.created")
			}
		}
	}
}

func handleCheckoutSessionCompleted(c *RequestContext, sc *stripe.Client, session *stripe.CheckoutSession) {
	if session.ClientReferenceID == "" {
		logging.Error().Msg("checkout.session.completed missing client_reference_id")
		return
	}

	userID, err := strconv.Atoi(session.ClientReferenceID)
	if err != nil {
		logging.Error().Err(err).Str("client_reference_id", session.ClientReferenceID).Msg("invalid client_reference_id")
		return
	}

	// Link user and update status. We use RETURNING to avoid a separate query for the email.
	user, err := db.QueryOne[models.User](c, c.Conn, `
		UPDATE hmn_user 
		SET 
			is_subscribed = true, 
			stripe_customer_id = $1, 
			stripe_subscription_id = $2, 
			subscription_status = 'active',
			cancel_at_period_end = false
		WHERE id = $3
		RETURNING $columns
	`, session.Customer.ID, session.Subscription.ID, userID)
	if err != nil {
		logging.Error().Err(err).Int("userID", userID).Msg("failed to update user subscription status")
	} else {
		logging.Info().Int("userID", userID).Msg("user subscription linked, attempting thank you email")
		attemptThankYouEmail(c, user.ID, session.AmountTotal, session.Currency)
	}
}

func handleSubscriptionUpdated(c *RequestContext, sc *stripe.Client, sub *stripe.Subscription) {
	renewalDate := getSubscriptionPeriodEnd(sub)

	// Fetch user. Try customer ID first, fall back to metadata if linked yet.
	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", sub.Customer.ID)
	if err == db.NotFound {
		if uidStr, ok := sub.Metadata["user_id"]; ok {
			if uid, subErr := strconv.Atoi(uidStr); subErr == nil {
				user, err = db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE id = $1", uid)
			}
		}
	}

	if err != nil {
		logging.Error().Err(err).Str("customerID", sub.Customer.ID).Msg("failed to fetch user for subscription update")
		return
	}

	isCancelling := sub.CancelAtPeriodEnd || (sub.CancelAt > 0 && sub.Status != "canceled")

	logging.Info().
		Int("userID", user.ID).
		Str("status", string(sub.Status)).
		Bool("cancelAtPeriodEnd", sub.CancelAtPeriodEnd).
		Int64("cancelAt", sub.CancelAt).
		Bool("isCancelling", isCancelling).
		Msg("updating user subscription from webhook")

	_, err = c.Conn.Exec(c, `
		UPDATE hmn_user 
		SET 
			stripe_customer_id = $1,
			stripe_subscription_id = $2,
			subscription_status = $3, 
			cancel_at_period_end = $4,
			is_subscribed = ($3 = 'active' OR $3 = 'trialing'),
			current_period_end = $5
		WHERE id = $6
	`, sub.Customer.ID, sub.ID, sub.Status, isCancelling, renewalDate, user.ID)
	if err != nil {
		logging.Error().Err(err).Int("userID", user.ID).Msg("failed to update user subscription from webhook")
	}

	if isCancelling && !user.CancelAtPeriodEnd {
		var expirationDate *time.Time
		if sub.CancelAt > 0 {
			t := time.Unix(sub.CancelAt, 0)
			expirationDate = &t
		} else if renewalDate != nil {
			expirationDate = renewalDate
		}
		logging.Info().Int("userID", user.ID).Msg("sending subscription cancellation initiation email")
		err = email.SendSubscriptionCancelledEmail(user.Email, user.BestName(), expirationDate, c.Perf)
		if err != nil {
			logging.Error().Err(err).Int("userID", user.ID).Msg("failed to send cancellation initiation email")
		}
	}
}

func handleSubscriptionDeleted(c *RequestContext, sc *stripe.Client, sub *stripe.Subscription) {
	// Fetch user first to get their info for the email and use ID for update
	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", sub.Customer.ID)
	if err == db.NotFound {
		if uidStr, ok := sub.Metadata["user_id"]; ok {
			if uid, subErr := strconv.Atoi(uidStr); subErr == nil {
				user, err = db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE id = $1", uid)
			}
		}
	}

	if err != nil {
		logging.Error().Err(err).Str("customerID", sub.Customer.ID).Msg("failed to fetch user for subscription deletion")
		return
	}

	_, err = c.Conn.Exec(c, "UPDATE hmn_user SET is_subscribed = false, stripe_subscription_id = NULL, subscription_status = 'canceled', current_period_end = NULL, cancel_at_period_end = false, thank_you_email_sent = false WHERE id = $1", user.ID)
	if err != nil {
		logging.Error().Err(err).Int("userID", user.ID).Msg("failed to handle subscription deletion")
		return
	}

	logging.Info().Int("userID", user.ID).Msg("user subscription deactivated")

	// Only send cancellation email here if it wasn't already handled by cancelAtPeriodEnd initiation
	if !sub.CancelAtPeriodEnd {
		err = email.SendSubscriptionCancelledEmail(user.Email, user.BestName(), nil, c.Perf)
		if err != nil {
			logging.Error().Err(err).Int("userID", user.ID).Msg("failed to send cancellation email")
		}
	}
}

func handleInvoicePaid(c *RequestContext, sc *stripe.Client, inv *stripe.Invoice) {
	if inv.Customer == nil {
		return
	}

	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", inv.Customer.ID)
	if err == db.NotFound {
		// Future: could check metadata here if we expand sub or cust, but for now we'll wait for linking
	}
	if err != nil {
		logging.Error().Err(err).Str("customerID", inv.Customer.ID).Msg("failed to fetch user for invoice.paid")
		return
	}

	// Record payment details
	details := getPaymentDetails(c, sc, inv)

	_, err = c.Conn.Exec(c, `
		INSERT INTO user_payment (user_id, stripe_invoice_id, amount_cents, currency, payment_method_type, card_last4, card_brand, paid_at, stripe_fee_cents, net_amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (stripe_invoice_id) DO UPDATE SET 
			payment_method_type = EXCLUDED.payment_method_type,
			card_last4 = EXCLUDED.card_last4,
			card_brand = EXCLUDED.card_brand,
			stripe_fee_cents = EXCLUDED.stripe_fee_cents,
			net_amount_cents = EXCLUDED.net_amount_cents
	`, user.ID, inv.ID, inv.AmountPaid, string(inv.Currency), details.methodType, details.last4, details.brand, time.Unix(inv.StatusTransitions.PaidAt, 0), details.feeCents, details.netCents)
	if err != nil {
		logging.Error().Err(err).Int("userID", user.ID).Msg("failed to record user payment")
	}

	if inv.Lines != nil && len(inv.Lines.Data) > 0 && inv.Lines.Data[0].Subscription != nil {
		sub, err := sc.V1Subscriptions.Retrieve(c, inv.Lines.Data[0].Subscription.ID, nil)
		if err == nil {
			renewalDate := getSubscriptionPeriodEnd(sub)
			_, err = c.Conn.Exec(c, "UPDATE hmn_user SET current_period_end = $1, is_subscribed = true WHERE id = $2", renewalDate, user.ID)
			if err != nil {
				logging.Error().Err(err).Int("userID", user.ID).Msg("failed to update renewal date from invoice")
			}
		}
	}

	attemptThankYouEmail(c, user.ID, inv.AmountPaid, inv.Currency)
}

func handleInvoicePaymentFailed(c *RequestContext, sc *stripe.Client, inv *stripe.Invoice) {
	if inv.Customer == nil {
		return
	}

	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", inv.Customer.ID)
	if err != nil {
		logging.Error().Err(err).Str("customerID", inv.Customer.ID).Msg("failed to fetch user for invoice.payment_failed")
		return
	}

	amountStr := ""
	if inv.AmountDue > 0 {
		curr := strings.ToUpper(string(inv.Currency))
		symbol := "$"
		if curr != "USD" {
			symbol = curr + " "
		}
		amountStr = fmt.Sprintf("%s%.2f", symbol, float64(inv.AmountDue)/100.0)
	}

	var nextAttemptDate *time.Time
	if inv.NextPaymentAttempt > 0 {
		t := time.Unix(inv.NextPaymentAttempt, 0)
		nextAttemptDate = &t
	}

	logging.Info().Int("userID", user.ID).Str("invoiceID", inv.ID).Msg("sending payment failed email")
	err = email.SendPaymentFailedEmail(user.Email, user.BestName(), amountStr, nextAttemptDate, c.Perf)
	if err != nil {
		logging.Error().Err(err).Int("userID", user.ID).Msg("failed to send payment failed email")
	}
}

func getSubscriptionPeriodEnd(sub *stripe.Subscription) *time.Time {
	if sub != nil && sub.Items != nil && len(sub.Items.Data) > 0 {
		t := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
		return &t
	}
	return nil
}

type paymentDetails struct {
	methodType *string
	last4      *string
	brand      *string
	feeCents   *int
	netCents   *int
}

func getPaymentDetails(c context.Context, sc *stripe.Client, inv *stripe.Invoice) paymentDetails {
	var details paymentDetails
	params := &stripe.InvoicePaymentListParams{
		Invoice: stripe.String(inv.ID),
	}
	params.AddExpand("data.payment.charge.balance_transaction")
	params.AddExpand("data.payment.payment_intent.latest_charge")

	sc.V1InvoicePayments.List(c, params)(func(ip *stripe.InvoicePayment, err error) bool {
		if err != nil || ip.Payment == nil {
			return true
		}

		var targetCharge *stripe.Charge
		if ip.Payment.Charge != nil {
			targetCharge = ip.Payment.Charge
		} else if ip.Payment.PaymentIntent != nil && ip.Payment.PaymentIntent.LatestCharge != nil {
			targetCharge = ip.Payment.PaymentIntent.LatestCharge
		}

		if targetCharge == nil {
			return true
		}

		if targetCharge.PaymentMethodDetails != nil {
			mt := string(targetCharge.PaymentMethodDetails.Type)
			details.methodType = &mt
			if targetCharge.PaymentMethodDetails.Card != nil {
				l4 := targetCharge.PaymentMethodDetails.Card.Last4
				details.last4 = &l4
				b := string(targetCharge.PaymentMethodDetails.Card.Brand)
				details.brand = &b
			} else if targetCharge.PaymentMethodDetails.USBankAccount != nil {
				l4 := targetCharge.PaymentMethodDetails.USBankAccount.Last4
				details.last4 = &l4
				b := targetCharge.PaymentMethodDetails.USBankAccount.BankName
				details.brand = &b
			}
		}

		bt := targetCharge.BalanceTransaction
		if bt == nil || (bt.Net == 0 && inv.AmountPaid != 0) {
			retrieveParams := &stripe.ChargeRetrieveParams{}
			retrieveParams.AddExpand("balance_transaction")
			fullCharge, err := sc.V1Charges.Retrieve(c, targetCharge.ID, retrieveParams)
			if err == nil && fullCharge.BalanceTransaction != nil && (fullCharge.BalanceTransaction.Net != 0 || inv.AmountPaid == 0) {
				bt = fullCharge.BalanceTransaction
			} else {
				btListParams := &stripe.BalanceTransactionListParams{
					Source: stripe.String(targetCharge.ID),
				}
				sc.V1BalanceTransactions.List(c, btListParams)(func(item *stripe.BalanceTransaction, err error) bool {
					if err == nil && (item.Net != 0 || inv.AmountPaid == 0) {
						bt = item
						return false
					}
					return true
				})
			}
		}

		if bt != nil {
			fc := int(bt.Fee)
			details.feeCents = &fc
			nc := int(bt.Net)
			details.netCents = &nc
		}

		return false // Found a payment, stop iteration
	})

	return details
}

func attemptThankYouEmail(c *RequestContext, userID int, amountCents int64, currency stripe.Currency) {
	tx, err := c.Conn.Begin(c)
	if err != nil {
		logging.Error().Err(err).Int("userID", userID).Msg("failed to begin transaction for thank you email")
		return
	}
	defer tx.Rollback(c)

	user, err := db.QueryOne[models.User](c, tx, "SELECT $columns FROM hmn_user WHERE id = $1 FOR UPDATE", userID)
	if err != nil {
		if err != db.NotFound {
			logging.Error().Err(err).Int("userID", userID).Msg("failed to query user for thank you email")
		}
		return
	}

	// Only send if we have both pieces of info (active status and renewal date) and haven't sent it yet.
	// We check that the renewal date is at least reasonably in the future to avoid race conditions
	// where an old or "initiation" date is still in the DB.
	shouldSend := false
	if user.IsSubscribed && user.CurrentPeriodEnd != nil && user.CurrentPeriodEnd.After(time.Now().Add(24*time.Hour)) && !user.ThankYouEmailSent {
		shouldSend = true
		_, err = tx.Exec(c, "UPDATE hmn_user SET thank_you_email_sent = true WHERE id = $1", userID)
		if err != nil {
			logging.Error().Err(err).Int("userID", userID).Msg("failed to update thank_you_email_sent flag")
			return
		}
	}

	err = tx.Commit(c)
	if err != nil {
		logging.Error().Err(err).Int("userID", userID).Msg("failed to commit transaction for thank you email")
		return
	}

	if shouldSend {
		sendThankYouEmail(c, user, user.CurrentPeriodEnd, amountCents, currency)
	}
}

func sendThankYouEmail(c *RequestContext, user *models.User, renewalDate *time.Time, amountCents int64, currency stripe.Currency) {
	amountStr := ""
	if amountCents > 0 {
		curr := strings.ToUpper(string(currency))
		symbol := "$"
		if curr != "USD" {
			symbol = curr + " "
		}
		amountStr = fmt.Sprintf("%s%.2f", symbol, float64(amountCents)/100.0)
	}

	err := email.SendThankYouEmail(user.Email, user.BestName(), renewalDate, amountStr, c.Perf)
	if err != nil {
		logging.Error().Err(err).Int("userID", user.ID).Msg("failed to send thank you email")
	}
}
