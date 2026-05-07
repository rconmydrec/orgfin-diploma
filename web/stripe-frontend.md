# Stripe Frontend Integration Plan

## Overview

Integration of Stripe Checkout on the Vue.js frontend to handle premium subscription payments. The frontend will redirect users to Stripe Checkout for payment processing and handle the return flow.

## Backend API Endpoints (already implemented)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payments/checkout` | Create Checkout Session, returns URL |
| POST | `/payments/portal` | Create Customer Portal session |
| GET | `/payments/status` | Get payment/subscription status |
| GET | `/payments/session/{id}` | Get checkout session status |
| GET | `/payments/upgrade-price/{plan_id}` | Get price with credit info |
| POST | `/payments/webhook` | Stripe webhook (backend only) |

---

## Phase 1: Payment Service

### 1.1 Create `src/services/paymentService.js`

```javascript
// Methods:
- createCheckoutSession(planId) → { checkoutUrl, sessionId }
- createPortalSession() → { portalUrl }
- getPaymentStatus() → { status, subscription }
- getCheckoutSessionStatus(sessionId) → { status, paymentStatus }
- getUpgradePrice(planId) → { originalPrice, adjustedPrice, creditAmount, creditDays }
```

---

## Phase 2: Payment Store

### 2.1 Create `src/stores/payment.js`

```javascript
// State:
- checkoutSessionId: string | null
- isProcessingPayment: boolean
- paymentError: string | null
- lastPaymentStatus: object | null

// Actions:
- initiateCheckout(planId) → redirects to Stripe
- checkPaymentStatus(sessionId)
- openCustomerPortal()
- clearPaymentState()
```

---

## Phase 3: Payment Views

### 3.1 Create `src/views/PaymentSuccessView.vue`

- Route: `/payment/success?session_id=xxx`
- Display success message
- Verify payment via `getCheckoutSessionStatus()`
- Refresh subscription data
- Redirect to home/settings after delay

### 3.2 Create `src/views/PaymentCancelView.vue`

- Route: `/payment/cancel`
- Display cancellation message
- Option to retry or go back

---

## Phase 4: Update Subscription Components

### 4.1 Modify `SubscriptionUpgradeModal.vue`

**Current flow:**
```
User selects plan → API call upgradeToPremium() → Subscription updated
```

**New flow:**
```
User selects plan → API call createCheckoutSession() → Redirect to Stripe →
Webhook activates subscription → User redirected to /payment/success
```

Changes:
- [ ] Replace `handleUpgrade()` to call `createCheckoutSession()` instead of `upgradeToPremium()`
- [ ] Show loading state while redirecting to Stripe
- [ ] Keep downgrade flow as-is (no payment needed)
- [ ] Show upgrade price preview with credit calculation

### 4.2 Modify `SubscriptionBadge.vue`

- [ ] Add "Manage Subscription" button for premium users
- [ ] Opens Stripe Customer Portal

---

## Phase 5: Router Updates

### 5.1 Add routes in `src/router/index.js`

```javascript
{
  path: '/payment/success',
  name: 'PaymentSuccess',
  component: () => import('@/views/PaymentSuccessView.vue'),
  meta: { requiresAuth: true }
},
{
  path: '/payment/cancel',
  name: 'PaymentCancel',
  component: () => import('@/views/PaymentCancelView.vue'),
  meta: { requiresAuth: true }
}
```

---

## Phase 6: Translations

### 6.1 Add to `src/locales/en.json`

```json
{
  "payment": {
    "processing": "Processing payment...",
    "success": "Payment successful!",
    "successDescription": "Your premium subscription is now active.",
    "cancelled": "Payment cancelled",
    "cancelledDescription": "Your payment was cancelled. You can try again anytime.",
    "tryAgain": "Try Again",
    "goBack": "Go Back",
    "redirectingToStripe": "Redirecting to secure payment...",
    "manageBilling": "Manage Billing",
    "verifyingPayment": "Verifying payment...",
    "error": "Payment error",
    "sessionExpired": "Payment session expired"
  }
}
```

### 6.2 Add to `src/locales/uk.json`

```json
{
  "payment": {
    "processing": "Обробка платежу...",
    "success": "Платіж успішний!",
    "successDescription": "Вашу преміум підписку активовано.",
    "cancelled": "Платіж скасовано",
    "cancelledDescription": "Ваш платіж було скасовано. Ви можете спробувати ще раз.",
    "tryAgain": "Спробувати знову",
    "goBack": "Повернутися",
    "redirectingToStripe": "Перенаправлення на безпечну оплату...",
    "manageBilling": "Керувати оплатою",
    "verifyingPayment": "Перевірка платежу...",
    "error": "Помилка платежу",
    "sessionExpired": "Сесія платежу закінчилась"
  }
}
```

---

## Phase 7: Settings Page Update

### 7.1 Modify `src/views/Settings.vue`

- [ ] Add "Manage Billing" section for premium users
- [ ] Button to open Stripe Customer Portal
- [ ] Show subscription details (plan, renewal date, etc.)

---

## Implementation Order

1. **Phase 1**: Payment Service (API layer)
2. **Phase 2**: Payment Store (state management)
3. **Phase 3**: Payment Views (success/cancel pages)
4. **Phase 5**: Router Updates
5. **Phase 4**: Update Subscription Components
6. **Phase 6**: Translations
7. **Phase 7**: Settings Page Update

---

## File Structure After Implementation

```
src/
├── services/
│   ├── paymentService.js     # NEW
│   └── subscriptionService.js # EXISTING
├── stores/
│   ├── payment.js            # NEW
│   └── subscription.js       # EXISTING (minor updates)
├── views/
│   ├── PaymentSuccessView.vue # NEW
│   ├── PaymentCancelView.vue  # NEW
│   └── Settings.vue           # UPDATE
├── components/
│   └── subscription/
│       ├── SubscriptionUpgradeModal.vue # UPDATE
│       └── SubscriptionBadge.vue        # UPDATE
├── router/
│   └── index.js               # UPDATE
└── locales/
    ├── en.json                # UPDATE
    └── uk.json                # UPDATE
```

---

## User Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           UPGRADE FLOW                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. User clicks "Upgrade" in SubscriptionUpgradeModal                   │
│                    ↓                                                     │
│  2. Frontend calls POST /payments/checkout { planId }                   │
│                    ↓                                                     │
│  3. Backend creates Stripe Checkout Session                             │
│     - Creates/gets Stripe Customer                                      │
│     - Applies credit from current subscription                          │
│     - Returns checkout URL                                               │
│                    ↓                                                     │
│  4. Frontend redirects to Stripe Checkout URL                           │
│                    ↓                                                     │
│  5. User completes payment on Stripe                                    │
│                    ↓                                                     │
│  6. Stripe redirects to /payment/success?session_id=xxx                 │
│                    ↓                                                     │
│  7. PaymentSuccessView verifies payment status                          │
│                    ↓                                                     │
│  8. Stripe webhook (checkout.session.completed) activates subscription  │
│                    ↓                                                     │
│  9. Frontend refreshes subscription data and shows success              │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                        MANAGE BILLING FLOW                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Premium user clicks "Manage Billing" in Settings                    │
│                    ↓                                                     │
│  2. Frontend calls POST /payments/portal                                │
│                    ↓                                                     │
│  3. Backend creates Stripe Customer Portal Session                      │
│                    ↓                                                     │
│  4. Frontend redirects to Stripe Customer Portal                        │
│                    ↓                                                     │
│  5. User can:                                                           │
│     - Update payment method                                              │
│     - View invoices                                                      │
│     - Cancel subscription                                                │
│                    ↓                                                     │
│  6. User returns to app                                                  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Notes

- **No Stripe.js SDK needed** - using Checkout Sessions (redirect mode)
- **Credit calculation** - backend handles credit from remaining subscription days
- **Webhook handles activation** - frontend just verifies and refreshes data
- **Downgrade flow unchanged** - no payment needed, stays in-app
- **Customer Portal** - for billing management (payment methods, invoices, cancel)
