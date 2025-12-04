// Package api - UI Handler for dynamic pages
package api

import (
	"net/http"

	"github.com/aethra/genesis/internal/models"
	"github.com/aethra/genesis/internal/ui"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UIHandler handles dynamic UI rendering
type UIHandler struct {
	db       *gorm.DB
	renderer *ui.Renderer
}

// NewUIHandler creates a new UI handler
func NewUIHandler(db *gorm.DB) *UIHandler {
	return &UIHandler{
		db:       db,
		renderer: ui.NewRenderer(db),
	}
}

// AppPage serves the dynamic app page
func (h *UIHandler) AppPage(c *gin.Context) {
	// Check if tenant exists
	var tenant models.Tenant
	if err := h.db.First(&tenant).Error; err != nil {
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	// Get user from session/token
	user := h.getUserFromContext(c)

	// Get the full path
	path := c.Request.URL.Path

	// Render page
	html, err := h.renderer.RenderPage(tenant.ID, path, user)
	if err != nil {
		// Log error and show simple error page
		c.String(http.StatusInternalServerError, "Error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// getUserFromContext extracts user info from JWT token
func (h *UIHandler) getUserFromContext(c *gin.Context) map[string]any {
	user := make(map[string]any)

	// Try to get user ID from context (set by auth middleware)
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uuid.UUID); ok {
			var dbUser models.User
			if err := h.db.First(&dbUser, "id = ?", uid).Error; err == nil {
				user["id"] = dbUser.ID
				user["email"] = dbUser.Email
				user["first_name"] = dbUser.FirstName
				user["last_name"] = dbUser.LastName
			}
		}
	}

	return user
}

// LoginPage serves the login page for the app
func (h *UIHandler) LoginPage(c *gin.Context) {
	// Check if tenant exists
	var tenant models.Tenant
	if err := h.db.First(&tenant).Error; err != nil {
		c.Redirect(http.StatusFound, "/setup")
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - Genesis</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #fff;
        }
        .login-box {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 16px;
            padding: 40px;
            width: 100%;
            max-width: 400px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo h1 {
            font-size: 32px;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .logo p {
            color: rgba(255,255,255,0.5);
            margin-top: 8px;
        }
        .form-group { margin-bottom: 20px; }
        .form-label {
            display: block;
            font-size: 13px;
            color: rgba(255,255,255,0.7);
            margin-bottom: 8px;
        }
        .form-input {
            width: 100%;
            padding: 12px 16px;
            background: rgba(255,255,255,0.05);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 8px;
            color: #fff;
            font-size: 14px;
        }
        .form-input:focus { outline: none; border-color: #00d4ff; }
        .btn {
            width: 100%;
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            color: #fff;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,212,255,0.3);
        }
        .error {
            background: rgba(239,68,68,0.1);
            border: 1px solid rgba(239,68,68,0.3);
            color: #ef4444;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: none;
        }
    </style>
</head>
<body>
    <div class="login-box">
        <div class="logo">
            <h1>Genesis</h1>
            <p>` + tenant.Name + `</p>
        </div>
        <div class="error" id="error"></div>
        <div class="form-group">
            <label class="form-label">Email</label>
            <input type="email" class="form-input" id="email" placeholder="admin@example.com">
        </div>
        <div class="form-group">
            <label class="form-label">Password</label>
            <input type="password" class="form-input" id="password" placeholder="Your password">
        </div>
        <button class="btn" onclick="doLogin()">Sign In</button>
    </div>

    <script>
        const TENANT_ID = '` + tenant.ID.String() + `';

        async function doLogin() {
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            const errorEl = document.getElementById('error');
            errorEl.style.display = 'none';

            try {
                const res = await fetch('/auth/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({email, password, tenant_id: TENANT_ID})
                });
                const data = await res.json();

                if (!res.ok) throw new Error(data.error || 'Login failed');

                const token = data.tokens?.access_token || data.token;
                localStorage.setItem('token', token);
                localStorage.setItem('user', JSON.stringify(data.user));
                localStorage.setItem('tenant_id', TENANT_ID);

                window.location.href = '/app';
            } catch (err) {
                errorEl.textContent = err.message;
                errorEl.style.display = 'block';
            }
        }

        document.getElementById('password').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') doLogin();
        });

        // Check if already logged in
        if (localStorage.getItem('token')) {
            window.location.href = '/app';
        }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
