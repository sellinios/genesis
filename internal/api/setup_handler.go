// Package api - Setup wizard handler
package api

import (
	"fmt"
	"net/http"

	"github.com/aethra/genesis/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SetupHandler handles installation wizard
type SetupHandler struct {
	db *gorm.DB
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(db *gorm.DB) *SetupHandler {
	return &SetupHandler{db: db}
}

// SetupPage serves the setup wizard HTML
func (h *SetupHandler) SetupPage(c *gin.Context) {
	// Check if already setup (any tenant exists)
	var count int64
	h.db.Model(&models.Tenant{}).Count(&count)

	if count > 0 {
		// Redirect to admin panel if setup is complete
		c.Redirect(http.StatusFound, "/panel")
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Genesis Setup</title>
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
        .container {
            background: rgba(255,255,255,0.05);
            border-radius: 16px;
            padding: 40px;
            width: 100%;
            max-width: 480px;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,0.1);
        }
        h1 {
            text-align: center;
            margin-bottom: 8px;
            font-size: 28px;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .subtitle {
            text-align: center;
            color: rgba(255,255,255,0.6);
            margin-bottom: 32px;
            font-size: 14px;
        }
        .step {
            display: none;
        }
        .step.active {
            display: block;
        }
        .step-title {
            font-size: 18px;
            margin-bottom: 20px;
            color: #00d4ff;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: rgba(255,255,255,0.8);
            font-size: 14px;
        }
        input {
            width: 100%;
            padding: 12px 16px;
            border: 1px solid rgba(255,255,255,0.2);
            border-radius: 8px;
            background: rgba(255,255,255,0.05);
            color: #fff;
            font-size: 16px;
            transition: border-color 0.3s;
        }
        input:focus {
            outline: none;
            border-color: #00d4ff;
        }
        input::placeholder {
            color: rgba(255,255,255,0.3);
        }
        .btn {
            width: 100%;
            padding: 14px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
        }
        .btn-primary {
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            color: #fff;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,212,255,0.3);
        }
        .btn-primary:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none;
        }
        .progress {
            display: flex;
            justify-content: center;
            gap: 8px;
            margin-bottom: 32px;
        }
        .progress-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: rgba(255,255,255,0.2);
            transition: all 0.3s;
        }
        .progress-dot.active {
            background: #00d4ff;
        }
        .progress-dot.done {
            background: #7b2cbf;
        }
        .error {
            background: rgba(255,59,48,0.2);
            border: 1px solid rgba(255,59,48,0.5);
            color: #ff6b6b;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: none;
        }
        .success {
            text-align: center;
            padding: 40px 20px;
        }
        .success-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        .success h2 {
            margin-bottom: 16px;
        }
        .success p {
            color: rgba(255,255,255,0.6);
            margin-bottom: 24px;
        }
        .loader {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid rgba(255,255,255,0.3);
            border-radius: 50%;
            border-top-color: #fff;
            animation: spin 1s ease-in-out infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Genesis</h1>
        <p class="subtitle">Initial Setup</p>

        <div class="progress">
            <div class="progress-dot active" id="dot1"></div>
            <div class="progress-dot" id="dot2"></div>
            <div class="progress-dot" id="dot3"></div>
        </div>

        <div class="error" id="error"></div>

        <!-- Step 1: Tenant -->
        <div class="step active" id="step1">
            <div class="step-title">1. Create Tenant</div>
            <div class="form-group">
                <label>Tenant Code</label>
                <input type="text" id="tenantCode" placeholder="e.g., acme" pattern="[a-z0-9-]+" required>
            </div>
            <div class="form-group">
                <label>Tenant Name</label>
                <input type="text" id="tenantName" placeholder="e.g., Acme Corporation" required>
            </div>
            <button class="btn btn-primary" onclick="nextStep(1)">Continue</button>
        </div>

        <!-- Step 2: Admin User -->
        <div class="step" id="step2">
            <div class="step-title">2. Create Admin User</div>
            <div class="form-group">
                <label>Email</label>
                <input type="email" id="adminEmail" placeholder="admin@example.com" required>
            </div>
            <div class="form-group">
                <label>Password</label>
                <input type="password" id="adminPassword" placeholder="Minimum 8 characters" required>
            </div>
            <div class="form-group">
                <label>First Name</label>
                <input type="text" id="firstName" placeholder="John" required>
            </div>
            <div class="form-group">
                <label>Last Name</label>
                <input type="text" id="lastName" placeholder="Doe" required>
            </div>
            <button class="btn btn-primary" onclick="nextStep(2)">Continue</button>
        </div>

        <!-- Step 3: Confirm -->
        <div class="step" id="step3">
            <div class="step-title">3. Confirm Setup</div>
            <div style="background: rgba(255,255,255,0.05); padding: 20px; border-radius: 8px; margin-bottom: 20px;">
                <p style="margin-bottom: 12px;"><strong>Tenant:</strong> <span id="confirmTenant"></span></p>
                <p><strong>Admin:</strong> <span id="confirmAdmin"></span></p>
            </div>
            <button class="btn btn-primary" id="setupBtn" onclick="doSetup()">Complete Setup</button>
        </div>

        <!-- Success -->
        <div class="step" id="step4">
            <div class="success">
                <div class="success-icon">✓</div>
                <h2>Setup Complete!</h2>
                <p>Genesis is ready. Redirecting to Dashboard...</p>
                <p id="countdown" style="color: #00d4ff; margin-top: 10px;">Redirecting in 3 seconds...</p>
            </div>
        </div>
    </div>

    <script>
        let currentStep = 1;
        let data = {};

        function showError(msg) {
            const el = document.getElementById('error');
            el.textContent = msg;
            el.style.display = 'block';
        }

        function hideError() {
            document.getElementById('error').style.display = 'none';
        }

        function nextStep(step) {
            hideError();

            if (step === 1) {
                data.tenantCode = document.getElementById('tenantCode').value.trim().toLowerCase();
                data.tenantName = document.getElementById('tenantName').value.trim();

                if (!data.tenantCode || !data.tenantName) {
                    showError('Please fill in all fields');
                    return;
                }
                if (!/^[a-z0-9-]+$/.test(data.tenantCode)) {
                    showError('Tenant code must be lowercase letters, numbers, and hyphens only');
                    return;
                }
            }

            if (step === 2) {
                data.email = document.getElementById('adminEmail').value.trim();
                data.password = document.getElementById('adminPassword').value;
                data.firstName = document.getElementById('firstName').value.trim();
                data.lastName = document.getElementById('lastName').value.trim();

                if (!data.email || !data.password || !data.firstName || !data.lastName) {
                    showError('Please fill in all fields');
                    return;
                }
                if (data.password.length < 8) {
                    showError('Password must be at least 8 characters');
                    return;
                }

                document.getElementById('confirmTenant').textContent = data.tenantName + ' (' + data.tenantCode + ')';
                document.getElementById('confirmAdmin').textContent = data.email;
            }

            document.getElementById('step' + step).classList.remove('active');
            document.getElementById('step' + (step + 1)).classList.add('active');
            document.getElementById('dot' + step).classList.remove('active');
            document.getElementById('dot' + step).classList.add('done');
            if (step < 3) {
                document.getElementById('dot' + (step + 1)).classList.add('active');
            }
            currentStep = step + 1;
        }

        async function doSetup() {
            hideError();
            const btn = document.getElementById('setupBtn');
            btn.disabled = true;
            btn.innerHTML = '<span class="loader"></span> Setting up...';

            try {
                const response = await fetch('/setup', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (!response.ok) {
                    throw new Error(result.error || 'Setup failed');
                }

                document.getElementById('step3').classList.remove('active');
                document.getElementById('step4').classList.add('active');
                document.getElementById('dot3').classList.remove('active');
                document.getElementById('dot3').classList.add('done');

                // Auto redirect to admin panel
                let seconds = 3;
                const countdownEl = document.getElementById('countdown');
                const interval = setInterval(() => {
                    seconds--;
                    if (seconds <= 0) {
                        clearInterval(interval);
                        window.location.href = '/panel';
                    } else {
                        countdownEl.textContent = 'Redirecting in ' + seconds + ' seconds...';
                    }
                }, 1000);

            } catch (err) {
                showError(err.message);
                btn.disabled = false;
                btn.textContent = 'Complete Setup';
            }
        }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// DoSetup processes the setup form
func (h *SetupHandler) DoSetup(c *gin.Context) {
	var input struct {
		TenantCode string `json:"tenantCode" binding:"required"`
		TenantName string `json:"tenantName" binding:"required"`
		Email      string `json:"email" binding:"required,email"`
		Password   string `json:"password" binding:"required,min=8"`
		FirstName  string `json:"firstName" binding:"required"`
		LastName   string `json:"lastName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if already setup
	var count int64
	h.db.Model(&models.Tenant{}).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setup already completed"})
		return
	}

	// Create tenant
	tenant := models.Tenant{
		ID:       uuid.New(),
		Code:     input.TenantCode,
		Name:     input.TenantName,
		IsActive: true,
	}

	if err := h.db.Create(&tenant).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create tenant: " + err.Error()})
		return
	}

	// Create admin user
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		ID:           uuid.New(),
		TenantID:     tenant.ID,
		Email:        input.Email,
		PasswordHash: string(hash),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		IsActive:     true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		// Rollback tenant
		h.db.Delete(&tenant)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	// Create super_admin role and assign to user
	role := models.Role{
		ID:          uuid.New(),
		TenantID:    tenant.ID,
		Code:        "super_admin",
		Name:        "Super Admin",
		Description: "Full system access",
		IsSystem:    true,
	}

	if err := h.db.Create(&role).Error; err != nil {
		// Log but don't fail - role might already exist
		fmt.Printf("Warning: Failed to create role: %v\n", err)
	} else {
		// Assign role to user via raw SQL (user_roles is a join table)
		if err := h.db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error; err != nil {
			fmt.Printf("Warning: Failed to assign role: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Setup complete",
		"tenant_id": tenant.ID,
		"user_id":   user.ID,
	})
}

// IsSetupRequired checks if setup is needed
func (h *SetupHandler) IsSetupRequired() bool {
	var count int64
	h.db.Model(&models.Tenant{}).Count(&count)
	return count == 0
}

// DashboardPage serves the main dashboard after setup
func (h *SetupHandler) DashboardPage(c *gin.Context) {
	// Get stats
	var tenantCount, userCount, moduleCount, entityCount int64
	h.db.Model(&models.Tenant{}).Count(&tenantCount)
	h.db.Model(&models.User{}).Count(&userCount)
	h.db.Model(&models.Module{}).Count(&moduleCount)
	h.db.Model(&models.Entity{}).Count(&entityCount)

	// Get tenant info
	var tenant models.Tenant
	h.db.First(&tenant)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Genesis Dashboard</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            color: #fff;
        }
        .header {
            background: rgba(255,255,255,0.05);
            padding: 20px 40px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .logo {
            font-size: 24px;
            font-weight: bold;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .tenant-badge {
            background: rgba(0,212,255,0.2);
            padding: 8px 16px;
            border-radius: 20px;
            font-size: 14px;
            color: #00d4ff;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 40px;
        }
        .welcome {
            margin-bottom: 40px;
        }
        .welcome h1 {
            font-size: 32px;
            margin-bottom: 8px;
        }
        .welcome p {
            color: rgba(255,255,255,0.6);
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .stat-card {
            background: rgba(255,255,255,0.05);
            border-radius: 12px;
            padding: 24px;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .stat-card h3 {
            font-size: 36px;
            margin-bottom: 8px;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .stat-card p {
            color: rgba(255,255,255,0.6);
            font-size: 14px;
        }
        .section {
            background: rgba(255,255,255,0.05);
            border-radius: 12px;
            padding: 24px;
            margin-bottom: 20px;
            border: 1px solid rgba(255,255,255,0.1);
        }
        .section h2 {
            font-size: 18px;
            margin-bottom: 16px;
            color: #00d4ff;
        }
        .api-list {
            display: grid;
            gap: 12px;
        }
        .api-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px;
            background: rgba(255,255,255,0.03);
            border-radius: 8px;
        }
        .method {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
            min-width: 60px;
            text-align: center;
        }
        .method.get { background: #10b981; color: #000; }
        .method.post { background: #3b82f6; color: #fff; }
        .method.put { background: #f59e0b; color: #000; }
        .method.delete { background: #ef4444; color: #fff; }
        .endpoint {
            font-family: monospace;
            color: rgba(255,255,255,0.8);
        }
        .login-form {
            max-width: 400px;
        }
        .form-group {
            margin-bottom: 16px;
        }
        .form-group label {
            display: block;
            margin-bottom: 8px;
            color: rgba(255,255,255,0.8);
            font-size: 14px;
        }
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 1px solid rgba(255,255,255,0.2);
            border-radius: 8px;
            background: rgba(255,255,255,0.05);
            color: #fff;
            font-size: 14px;
        }
        .form-group input:focus {
            outline: none;
            border-color: #00d4ff;
        }
        .btn {
            padding: 12px 24px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            color: #fff;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,212,255,0.3);
        }
        .token-display {
            margin-top: 16px;
            padding: 12px;
            background: rgba(0,0,0,0.3);
            border-radius: 8px;
            font-family: monospace;
            font-size: 12px;
            word-break: break-all;
            display: none;
        }
        .error {
            color: #ef4444;
            margin-top: 12px;
            display: none;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">Genesis</div>
        <div class="tenant-badge">` + tenant.Name + `</div>
    </div>

    <div class="container">
        <div class="welcome">
            <h1>Welcome to Genesis</h1>
            <p>Your dynamic business platform is ready</p>
        </div>

        <div class="stats">
            <div class="stat-card">
                <h3>` + fmt.Sprintf("%d", tenantCount) + `</h3>
                <p>Tenants</p>
            </div>
            <div class="stat-card">
                <h3>` + fmt.Sprintf("%d", userCount) + `</h3>
                <p>Users</p>
            </div>
            <div class="stat-card">
                <h3>` + fmt.Sprintf("%d", moduleCount) + `</h3>
                <p>Modules</p>
            </div>
            <div class="stat-card">
                <h3>` + fmt.Sprintf("%d", entityCount) + `</h3>
                <p>Entities</p>
            </div>
        </div>

        <div class="section">
            <h2>Login</h2>
            <div class="login-form">
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" id="email" placeholder="admin@example.com">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="password" placeholder="Your password">
                </div>
                <button class="btn" onclick="doLogin()">Login</button>
                <div class="error" id="error"></div>
                <div class="token-display" id="token"></div>
            </div>
        </div>

        <div class="section">
            <h2>API Endpoints</h2>
            <div class="api-list">
                <div class="api-item">
                    <span class="method post">POST</span>
                    <span class="endpoint">/auth/login</span>
                </div>
                <div class="api-item">
                    <span class="method get">GET</span>
                    <span class="endpoint">/admin/tenants</span>
                </div>
                <div class="api-item">
                    <span class="method get">GET</span>
                    <span class="endpoint">/admin/modules</span>
                </div>
                <div class="api-item">
                    <span class="method get">GET</span>
                    <span class="endpoint">/admin/entities</span>
                </div>
                <div class="api-item">
                    <span class="method get">GET</span>
                    <span class="endpoint">/admin/fields</span>
                </div>
                <div class="api-item">
                    <span class="method get">GET</span>
                    <span class="endpoint">/api/health</span>
                </div>
            </div>
        </div>
    </div>

    <script>
        const TENANT_ID = '` + tenant.ID.String() + `';

        async function doLogin() {
            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            const errorEl = document.getElementById('error');
            const tokenEl = document.getElementById('token');

            errorEl.style.display = 'none';
            tokenEl.style.display = 'none';

            try {
                const res = await fetch('/auth/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({email, password, tenant_id: TENANT_ID})
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || data.details || 'Login failed');

                const token = data.tokens?.access_token || data.token;
                tokenEl.innerHTML = '<strong>Welcome, ' + data.user.first_name + '!</strong><br><br><strong>Token:</strong><br><code style="font-size:10px;word-break:break-all;">' + token.substring(0, 50) + '...</code>';
                tokenEl.style.display = 'block';
                localStorage.setItem('token', token);
                localStorage.setItem('tenant_id', TENANT_ID);
                localStorage.setItem('user', JSON.stringify(data.user));
            } catch (err) {
                errorEl.textContent = err.message;
                errorEl.style.display = 'block';
            }
        }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
