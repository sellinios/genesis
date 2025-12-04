// Package api - Admin Panel UI
package api

import (
	"fmt"
	"net/http"

	"github.com/aethra/genesis/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminPanelHandler serves the admin panel UI
type AdminPanelHandler struct {
	db *gorm.DB
}

// NewAdminPanelHandler creates a new admin panel handler
func NewAdminPanelHandler(db *gorm.DB) *AdminPanelHandler {
	return &AdminPanelHandler{db: db}
}

// AdminPanel serves the main admin panel SPA
func (h *AdminPanelHandler) AdminPanel(c *gin.Context) {
	// Get tenant info
	var tenant models.Tenant
	h.db.First(&tenant)

	// Get stats
	var tenantCount, userCount, moduleCount, entityCount, fieldCount int64
	h.db.Model(&models.Tenant{}).Count(&tenantCount)
	h.db.Model(&models.User{}).Count(&userCount)
	h.db.Model(&models.Module{}).Count(&moduleCount)
	h.db.Model(&models.Entity{}).Count(&entityCount)
	h.db.Model(&models.Field{}).Count(&fieldCount)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Genesis Admin</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0f0f1a;
            min-height: 100vh;
            color: #fff;
        }
        .app {
            display: flex;
            min-height: 100vh;
        }

        /* Sidebar */
        .sidebar {
            width: 260px;
            background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%);
            border-right: 1px solid rgba(255,255,255,0.1);
            display: flex;
            flex-direction: column;
        }
        .sidebar-header {
            padding: 20px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        .logo {
            font-size: 24px;
            font-weight: bold;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .tenant-name {
            font-size: 12px;
            color: rgba(255,255,255,0.5);
            margin-top: 4px;
        }
        .nav {
            flex: 1;
            padding: 20px 0;
        }
        .nav-section {
            padding: 0 20px;
            margin-bottom: 20px;
        }
        .nav-section-title {
            font-size: 11px;
            text-transform: uppercase;
            color: rgba(255,255,255,0.4);
            margin-bottom: 10px;
            letter-spacing: 1px;
        }
        .nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 20px;
            color: rgba(255,255,255,0.7);
            text-decoration: none;
            cursor: pointer;
            transition: all 0.2s;
            border-left: 3px solid transparent;
        }
        .nav-item:hover {
            background: rgba(255,255,255,0.05);
            color: #fff;
        }
        .nav-item.active {
            background: rgba(0,212,255,0.1);
            color: #00d4ff;
            border-left-color: #00d4ff;
        }
        .nav-icon {
            width: 20px;
            text-align: center;
        }
        .sidebar-footer {
            padding: 20px;
            border-top: 1px solid rgba(255,255,255,0.1);
        }
        .user-info {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .user-avatar {
            width: 36px;
            height: 36px;
            border-radius: 50%;
            background: linear-gradient(135deg, #00d4ff, #7b2cbf);
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
        }
        .user-details {
            flex: 1;
        }
        .user-name {
            font-size: 14px;
            font-weight: 500;
        }
        .user-role {
            font-size: 11px;
            color: rgba(255,255,255,0.5);
        }
        .logout-btn {
            background: none;
            border: none;
            color: rgba(255,255,255,0.5);
            cursor: pointer;
            padding: 8px;
            border-radius: 4px;
        }
        .logout-btn:hover {
            background: rgba(255,255,255,0.1);
            color: #ef4444;
        }

        /* Main Content */
        .main {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        .header {
            padding: 20px 30px;
            background: rgba(255,255,255,0.02);
            border-bottom: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .page-title {
            font-size: 24px;
            font-weight: 600;
        }
        .header-actions {
            display: flex;
            gap: 12px;
        }
        .content {
            flex: 1;
            padding: 30px;
            overflow-y: auto;
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 12px;
            padding: 24px;
        }
        .stat-value {
            font-size: 36px;
            font-weight: 700;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .stat-label {
            font-size: 14px;
            color: rgba(255,255,255,0.5);
            margin-top: 4px;
        }

        /* Buttons */
        .btn {
            padding: 10px 20px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
            display: inline-flex;
            align-items: center;
            gap: 8px;
        }
        .btn-primary {
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            color: #fff;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 24px rgba(0,212,255,0.3);
        }
        .btn-secondary {
            background: rgba(255,255,255,0.1);
            color: #fff;
        }
        .btn-secondary:hover {
            background: rgba(255,255,255,0.15);
        }
        .btn-danger {
            background: #ef4444;
            color: #fff;
        }
        .btn-sm {
            padding: 6px 12px;
            font-size: 12px;
        }

        /* Tables */
        .table-container {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 12px;
            overflow: hidden;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 16px 20px;
            text-align: left;
            border-bottom: 1px solid rgba(255,255,255,0.05);
        }
        th {
            background: rgba(255,255,255,0.03);
            font-size: 12px;
            text-transform: uppercase;
            color: rgba(255,255,255,0.5);
            font-weight: 600;
            letter-spacing: 0.5px;
        }
        tr:hover td {
            background: rgba(255,255,255,0.02);
        }
        .badge {
            display: inline-block;
            padding: 4px 10px;
            border-radius: 20px;
            font-size: 11px;
            font-weight: 600;
        }
        .badge-blue { background: rgba(59,130,246,0.2); color: #3b82f6; }
        .badge-green { background: rgba(16,185,129,0.2); color: #10b981; }
        .badge-purple { background: rgba(139,92,246,0.2); color: #8b5cf6; }
        .badge-yellow { background: rgba(245,158,11,0.2); color: #f59e0b; }

        /* Modal */
        .modal-overlay {
            position: fixed;
            inset: 0;
            background: rgba(0,0,0,0.7);
            display: none;
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }
        .modal-overlay.active {
            display: flex;
        }
        .modal {
            background: #1a1a2e;
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 16px;
            width: 100%;
            max-width: 500px;
            max-height: 90vh;
            overflow-y: auto;
        }
        .modal-header {
            padding: 20px 24px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .modal-title {
            font-size: 18px;
            font-weight: 600;
        }
        .modal-close {
            background: none;
            border: none;
            color: rgba(255,255,255,0.5);
            font-size: 24px;
            cursor: pointer;
        }
        .modal-body {
            padding: 24px;
        }
        .modal-footer {
            padding: 16px 24px;
            border-top: 1px solid rgba(255,255,255,0.1);
            display: flex;
            justify-content: flex-end;
            gap: 12px;
        }

        /* Forms */
        .form-group {
            margin-bottom: 20px;
        }
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
        .form-input:focus {
            outline: none;
            border-color: #00d4ff;
        }
        .form-input::placeholder {
            color: rgba(255,255,255,0.3);
        }
        select.form-input {
            cursor: pointer;
        }

        /* Empty state */
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: rgba(255,255,255,0.5);
        }
        .empty-icon {
            font-size: 48px;
            margin-bottom: 16px;
        }
        .empty-title {
            font-size: 18px;
            color: #fff;
            margin-bottom: 8px;
        }
        .empty-text {
            margin-bottom: 20px;
        }

        /* Loading */
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px;
        }
        .spinner {
            width: 32px;
            height: 32px;
            border: 3px solid rgba(255,255,255,0.1);
            border-top-color: #00d4ff;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Toast notifications */
        .toast-container {
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 2000;
            display: flex;
            flex-direction: column;
            gap: 10px;
        }
        .toast {
            padding: 16px 20px;
            background: #1a1a2e;
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 8px;
            display: flex;
            align-items: center;
            gap: 12px;
            animation: slideIn 0.3s ease;
        }
        .toast.success { border-left: 4px solid #10b981; }
        .toast.error { border-left: 4px solid #ef4444; }
        @keyframes slideIn {
            from { transform: translateX(100%); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }

        /* Login Page */
        .login-container {
            display: none;
            min-height: 100vh;
            align-items: center;
            justify-content: center;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
        }
        .login-container.active {
            display: flex;
        }
        .login-box {
            background: rgba(255,255,255,0.03);
            border: 1px solid rgba(255,255,255,0.1);
            border-radius: 16px;
            padding: 40px;
            width: 100%;
            max-width: 400px;
        }
        .login-logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .login-logo h1 {
            font-size: 32px;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .login-error {
            background: rgba(239,68,68,0.1);
            border: 1px solid rgba(239,68,68,0.3);
            color: #ef4444;
            padding: 12px;
            border-radius: 8px;
            margin-bottom: 20px;
            display: none;
        }

        /* Hide app when not logged in */
        .app { display: none; }
        .app.active { display: flex; }
    </style>
</head>
<body>
    <!-- Toast Container -->
    <div class="toast-container" id="toastContainer"></div>

    <!-- Login Page -->
    <div class="login-container" id="loginPage">
        <div class="login-box">
            <div class="login-logo">
                <h1>Genesis</h1>
                <p style="color: rgba(255,255,255,0.5); margin-top: 8px;">Admin Panel</p>
            </div>
            <div class="login-error" id="loginError"></div>
            <div class="form-group">
                <label class="form-label">Email</label>
                <input type="email" class="form-input" id="loginEmail" placeholder="admin@example.com">
            </div>
            <div class="form-group">
                <label class="form-label">Password</label>
                <input type="password" class="form-input" id="loginPassword" placeholder="Your password">
            </div>
            <button class="btn btn-primary" style="width: 100%;" onclick="doLogin()">Sign In</button>
        </div>
    </div>

    <!-- Main App -->
    <div class="app" id="app">
        <!-- Sidebar -->
        <div class="sidebar">
            <div class="sidebar-header">
                <div class="logo">Genesis</div>
                <div class="tenant-name" id="tenantName">` + tenant.Name + `</div>
            </div>
            <nav class="nav">
                <div class="nav-section">
                    <div class="nav-section-title">Overview</div>
                    <div class="nav-item active" onclick="showPage('dashboard')">
                        <span class="nav-icon">📊</span>
                        Dashboard
                    </div>
                </div>
                <div class="nav-section">
                    <div class="nav-section-title">Configuration</div>
                    <div class="nav-item" onclick="showPage('modules')">
                        <span class="nav-icon">📦</span>
                        Modules
                    </div>
                    <div class="nav-item" onclick="showPage('entities')">
                        <span class="nav-icon">🗃️</span>
                        Entities
                    </div>
                    <div class="nav-item" onclick="showPage('fields')">
                        <span class="nav-icon">📝</span>
                        Fields
                    </div>
                </div>
                <div class="nav-section">
                    <div class="nav-section-title">System</div>
                    <div class="nav-item" onclick="showPage('users')">
                        <span class="nav-icon">👥</span>
                        Users
                    </div>
                    <div class="nav-item" onclick="showPage('tenants')">
                        <span class="nav-icon">🏢</span>
                        Tenants
                    </div>
                </div>
            </nav>
            <div class="sidebar-footer">
                <div class="user-info">
                    <div class="user-avatar" id="userAvatar">A</div>
                    <div class="user-details">
                        <div class="user-name" id="userName">Admin</div>
                        <div class="user-role">Administrator</div>
                    </div>
                    <button class="logout-btn" onclick="doLogout()" title="Logout">⬅</button>
                </div>
            </div>
        </div>

        <!-- Main Content -->
        <div class="main">
            <div class="header">
                <h1 class="page-title" id="pageTitle">Dashboard</h1>
                <div class="header-actions" id="headerActions"></div>
            </div>
            <div class="content" id="content">
                <!-- Content loaded dynamically -->
            </div>
        </div>
    </div>

    <!-- Modal -->
    <div class="modal-overlay" id="modalOverlay" onclick="closeModal(event)">
        <div class="modal" onclick="event.stopPropagation()">
            <div class="modal-header">
                <h2 class="modal-title" id="modalTitle">Modal</h2>
                <button class="modal-close" onclick="closeModal()">&times;</button>
            </div>
            <div class="modal-body" id="modalBody"></div>
            <div class="modal-footer" id="modalFooter"></div>
        </div>
    </div>

    <script>
        // State
        const TENANT_ID = '` + tenant.ID.String() + `';
        let token = localStorage.getItem('token');
        let currentUser = JSON.parse(localStorage.getItem('user') || 'null');
        let currentPage = 'dashboard';

        // Initial stats from server
        const initialStats = {
            tenants: ` + fmt.Sprintf("%d", tenantCount) + `,
            users: ` + fmt.Sprintf("%d", userCount) + `,
            modules: ` + fmt.Sprintf("%d", moduleCount) + `,
            entities: ` + fmt.Sprintf("%d", entityCount) + `,
            fields: ` + fmt.Sprintf("%d", fieldCount) + `
        };

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            if (token && currentUser) {
                showApp();
            } else {
                showLogin();
            }
        });

        // Auth functions
        function showLogin() {
            document.getElementById('loginPage').classList.add('active');
            document.getElementById('app').classList.remove('active');
        }

        function showApp() {
            document.getElementById('loginPage').classList.remove('active');
            document.getElementById('app').classList.add('active');
            if (currentUser) {
                document.getElementById('userName').textContent = currentUser.first_name + ' ' + currentUser.last_name;
                document.getElementById('userAvatar').textContent = (currentUser.first_name || 'A')[0].toUpperCase();
            }
            showPage('dashboard');
        }

        async function doLogin() {
            const email = document.getElementById('loginEmail').value;
            const password = document.getElementById('loginPassword').value;
            const errorEl = document.getElementById('loginError');

            errorEl.style.display = 'none';

            try {
                const res = await fetch('/auth/login', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({email, password, tenant_id: TENANT_ID})
                });
                const data = await res.json();

                if (!res.ok) throw new Error(data.error || 'Login failed');

                token = data.tokens?.access_token || data.token;
                currentUser = data.user;
                localStorage.setItem('token', token);
                localStorage.setItem('user', JSON.stringify(currentUser));

                showApp();
                showToast('Welcome back, ' + currentUser.first_name + '!', 'success');
            } catch (err) {
                errorEl.textContent = err.message;
                errorEl.style.display = 'block';
            }
        }

        function doLogout() {
            token = null;
            currentUser = null;
            localStorage.removeItem('token');
            localStorage.removeItem('user');
            showLogin();
        }

        // API helper
        async function api(method, endpoint, body = null) {
            const options = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': 'Bearer ' + token,
                    'X-Tenant-ID': TENANT_ID
                }
            };
            if (body) options.body = JSON.stringify(body);

            const res = await fetch(endpoint, options);
            const data = await res.json();

            if (!res.ok) {
                if (res.status === 401) {
                    doLogout();
                    throw new Error('Session expired');
                }
                throw new Error(data.error || 'Request failed');
            }
            return data;
        }

        // Navigation
        function showPage(page) {
            currentPage = page;

            // Update nav
            document.querySelectorAll('.nav-item').forEach(item => {
                item.classList.remove('active');
                if (item.textContent.trim().toLowerCase().includes(page)) {
                    item.classList.add('active');
                }
            });

            // Render page
            const titles = {
                dashboard: 'Dashboard',
                modules: 'Modules',
                entities: 'Entities',
                fields: 'Fields',
                users: 'Users',
                tenants: 'Tenants'
            };
            document.getElementById('pageTitle').textContent = titles[page] || page;

            switch(page) {
                case 'dashboard': renderDashboard(); break;
                case 'modules': renderModules(); break;
                case 'entities': renderEntities(); break;
                case 'fields': renderFields(); break;
                case 'users': renderUsers(); break;
                case 'tenants': renderTenants(); break;
            }
        }

        // Dashboard
        function renderDashboard() {
            document.getElementById('headerActions').innerHTML = '';
            document.getElementById('content').innerHTML = ` + "`" + `
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-value">${initialStats.tenants}</div>
                        <div class="stat-label">Tenants</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${initialStats.users}</div>
                        <div class="stat-label">Users</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${initialStats.modules}</div>
                        <div class="stat-label">Modules</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${initialStats.entities}</div>
                        <div class="stat-label">Entities</div>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${initialStats.fields}</div>
                        <div class="stat-label">Fields</div>
                    </div>
                </div>
                <div class="table-container">
                    <div style="padding: 20px; border-bottom: 1px solid rgba(255,255,255,0.1);">
                        <h3>Quick Start Guide</h3>
                    </div>
                    <div style="padding: 20px;">
                        <ol style="padding-left: 20px; line-height: 2;">
                            <li>Create a <strong>Module</strong> (e.g., CRM, Sales, HR)</li>
                            <li>Add <strong>Entities</strong> to your module (e.g., Customers, Products)</li>
                            <li>Define <strong>Fields</strong> for each entity (e.g., name, email, price)</li>
                            <li>Use the <strong>API</strong> to manage your data</li>
                        </ol>
                    </div>
                </div>
            ` + "`" + `;
        }

        // Modules page
        async function renderModules() {
            document.getElementById('headerActions').innerHTML = '<button class="btn btn-primary" onclick="openCreateModuleModal()">+ New Module</button>';
            document.getElementById('content').innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const modules = await api('GET', '/admin/modules?tenant_id=' + TENANT_ID);

                if (modules.length === 0) {
                    document.getElementById('content').innerHTML = ` + "`" + `
                        <div class="empty-state">
                            <div class="empty-icon">📦</div>
                            <div class="empty-title">No modules yet</div>
                            <div class="empty-text">Modules group related entities together</div>
                            <button class="btn btn-primary" onclick="openCreateModuleModal()">Create First Module</button>
                        </div>
                    ` + "`" + `;
                    return;
                }

                document.getElementById('content').innerHTML = ` + "`" + `
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Code</th>
                                    <th>Description</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody id="modulesTable"></tbody>
                        </table>
                    </div>
                ` + "`" + `;

                const tbody = document.getElementById('modulesTable');
                modules.forEach(m => {
                    tbody.innerHTML += ` + "`" + `
                        <tr>
                            <td><span class="badge badge-blue">${m.icon || '📦'}</span> ${m.name}</td>
                            <td><code>${m.code}</code></td>
                            <td>${m.description || '-'}</td>
                            <td>
                                <button class="btn btn-secondary btn-sm" onclick="editModule('${m.id}')">Edit</button>
                                <button class="btn btn-danger btn-sm" onclick="deleteModule('${m.id}')">Delete</button>
                            </td>
                        </tr>
                    ` + "`" + `;
                });
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        function openCreateModuleModal() {
            document.getElementById('modalTitle').textContent = 'Create Module';
            document.getElementById('modalBody').innerHTML = ` + "`" + `
                <div class="form-group">
                    <label class="form-label">Name *</label>
                    <input type="text" class="form-input" id="moduleName" placeholder="e.g., CRM">
                </div>
                <div class="form-group">
                    <label class="form-label">Code *</label>
                    <input type="text" class="form-input" id="moduleCode" placeholder="e.g., crm (lowercase, no spaces)">
                </div>
                <div class="form-group">
                    <label class="form-label">Description</label>
                    <input type="text" class="form-input" id="moduleDescription" placeholder="Optional description">
                </div>
                <div class="form-group">
                    <label class="form-label">Icon</label>
                    <input type="text" class="form-input" id="moduleIcon" placeholder="e.g., 📦" value="📦">
                </div>
            ` + "`" + `;
            document.getElementById('modalFooter').innerHTML = ` + "`" + `
                <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
                <button class="btn btn-primary" onclick="createModule()">Create Module</button>
            ` + "`" + `;
            document.getElementById('modalOverlay').classList.add('active');
        }

        async function createModule() {
            const name = document.getElementById('moduleName').value.trim();
            const code = document.getElementById('moduleCode').value.trim().toLowerCase();
            const description = document.getElementById('moduleDescription').value.trim();
            const icon = document.getElementById('moduleIcon').value.trim();

            if (!name || !code) {
                showToast('Name and code are required', 'error');
                return;
            }

            try {
                await api('POST', '/admin/modules', {
                    tenant_id: TENANT_ID,
                    name, code, description, icon
                });
                closeModal();
                showToast('Module created successfully', 'success');
                renderModules();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        async function deleteModule(id) {
            if (!confirm('Are you sure you want to delete this module?')) return;
            try {
                await api('DELETE', '/admin/modules/' + id);
                showToast('Module deleted', 'success');
                renderModules();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // Entities page
        async function renderEntities() {
            document.getElementById('headerActions').innerHTML = '<button class="btn btn-primary" onclick="openCreateEntityModal()">+ New Entity</button>';
            document.getElementById('content').innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const entities = await api('GET', '/admin/entities');
                const modules = await api('GET', '/admin/modules?tenant_id=' + TENANT_ID);
                window.modulesCache = modules;

                if (entities.length === 0) {
                    document.getElementById('content').innerHTML = ` + "`" + `
                        <div class="empty-state">
                            <div class="empty-icon">🗃️</div>
                            <div class="empty-title">No entities yet</div>
                            <div class="empty-text">Entities define your data structures</div>
                            <button class="btn btn-primary" onclick="openCreateEntityModal()">Create First Entity</button>
                        </div>
                    ` + "`" + `;
                    return;
                }

                const moduleMap = {};
                modules.forEach(m => moduleMap[m.id] = m);

                document.getElementById('content').innerHTML = ` + "`" + `
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Code</th>
                                    <th>Module</th>
                                    <th>Table</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody id="entitiesTable"></tbody>
                        </table>
                    </div>
                ` + "`" + `;

                const tbody = document.getElementById('entitiesTable');
                entities.forEach(e => {
                    const mod = moduleMap[e.module_id] || {};
                    tbody.innerHTML += ` + "`" + `
                        <tr>
                            <td><span class="badge badge-green">${e.icon || '🗃️'}</span> ${e.name}</td>
                            <td><code>${e.code}</code></td>
                            <td>${mod.name || '-'}</td>
                            <td><code>${e.table_name}</code></td>
                            <td>
                                <button class="btn btn-secondary btn-sm" onclick="editEntity('${e.id}')">Edit</button>
                                <button class="btn btn-danger btn-sm" onclick="deleteEntity('${e.id}')">Delete</button>
                            </td>
                        </tr>
                    ` + "`" + `;
                });
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        function openCreateEntityModal() {
            const modules = window.modulesCache || [];
            const moduleOptions = modules.map(m => ` + "`" + `<option value="${m.id}">${m.name}</option>` + "`" + `).join('');

            document.getElementById('modalTitle').textContent = 'Create Entity';
            document.getElementById('modalBody').innerHTML = ` + "`" + `
                <div class="form-group">
                    <label class="form-label">Module *</label>
                    <select class="form-input" id="entityModule">
                        <option value="">Select module...</option>
                        ${moduleOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">Name *</label>
                    <input type="text" class="form-input" id="entityName" placeholder="e.g., Customer">
                </div>
                <div class="form-group">
                    <label class="form-label">Plural Name</label>
                    <input type="text" class="form-input" id="entityNamePlural" placeholder="e.g., Customers">
                </div>
                <div class="form-group">
                    <label class="form-label">Code *</label>
                    <input type="text" class="form-input" id="entityCode" placeholder="e.g., customer (lowercase, no spaces)">
                </div>
                <div class="form-group">
                    <label class="form-label">Description</label>
                    <input type="text" class="form-input" id="entityDescription" placeholder="Optional description">
                </div>
            ` + "`" + `;
            document.getElementById('modalFooter').innerHTML = ` + "`" + `
                <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
                <button class="btn btn-primary" onclick="createEntity()">Create Entity</button>
            ` + "`" + `;
            document.getElementById('modalOverlay').classList.add('active');
        }

        async function createEntity() {
            const module_id = document.getElementById('entityModule').value;
            const name = document.getElementById('entityName').value.trim();
            const name_plural = document.getElementById('entityNamePlural').value.trim();
            const code = document.getElementById('entityCode').value.trim().toLowerCase();
            const description = document.getElementById('entityDescription').value.trim();

            if (!module_id || !name || !code) {
                showToast('Module, name and code are required', 'error');
                return;
            }

            try {
                await api('POST', '/admin/entities', {
                    module_id, name, name_plural, code, description
                });
                closeModal();
                showToast('Entity created successfully', 'success');
                renderEntities();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        async function deleteEntity(id) {
            if (!confirm('Are you sure you want to delete this entity?')) return;
            try {
                await api('DELETE', '/admin/entities/' + id);
                showToast('Entity deleted', 'success');
                renderEntities();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // Fields page
        async function renderFields() {
            document.getElementById('headerActions').innerHTML = '<button class="btn btn-primary" onclick="openCreateFieldModal()">+ New Field</button>';
            document.getElementById('content').innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const fields = await api('GET', '/admin/fields');
                const entities = await api('GET', '/admin/entities');
                const fieldTypes = await api('GET', '/admin/field-types');
                window.entitiesCache = entities;
                window.fieldTypesCache = fieldTypes;

                if (fields.length === 0) {
                    document.getElementById('content').innerHTML = ` + "`" + `
                        <div class="empty-state">
                            <div class="empty-icon">📝</div>
                            <div class="empty-title">No fields yet</div>
                            <div class="empty-text">Fields define the attributes of your entities</div>
                            <button class="btn btn-primary" onclick="openCreateFieldModal()">Create First Field</button>
                        </div>
                    ` + "`" + `;
                    return;
                }

                const entityMap = {};
                entities.forEach(e => entityMap[e.id] = e);
                const typeMap = {};
                fieldTypes.forEach(t => typeMap[t.id] = t);

                document.getElementById('content').innerHTML = ` + "`" + `
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Code</th>
                                    <th>Entity</th>
                                    <th>Type</th>
                                    <th>Required</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody id="fieldsTable"></tbody>
                        </table>
                    </div>
                ` + "`" + `;

                const tbody = document.getElementById('fieldsTable');
                fields.forEach(f => {
                    const entity = entityMap[f.entity_id] || {};
                    const ftype = typeMap[f.field_type_id] || {};
                    tbody.innerHTML += ` + "`" + `
                        <tr>
                            <td>${f.name}</td>
                            <td><code>${f.code}</code></td>
                            <td>${entity.name || '-'}</td>
                            <td><span class="badge badge-purple">${ftype.name || f.field_type_id}</span></td>
                            <td>${f.is_required ? '✓' : '-'}</td>
                            <td>
                                <button class="btn btn-secondary btn-sm" onclick="editField('${f.id}')">Edit</button>
                                <button class="btn btn-danger btn-sm" onclick="deleteField('${f.id}')">Delete</button>
                            </td>
                        </tr>
                    ` + "`" + `;
                });
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        function openCreateFieldModal() {
            const entities = window.entitiesCache || [];
            const fieldTypes = window.fieldTypesCache || [];
            const entityOptions = entities.map(e => ` + "`" + `<option value="${e.id}">${e.name}</option>` + "`" + `).join('');
            const typeOptions = fieldTypes.map(t => ` + "`" + `<option value="${t.id}">${t.name}</option>` + "`" + `).join('');

            document.getElementById('modalTitle').textContent = 'Create Field';
            document.getElementById('modalBody').innerHTML = ` + "`" + `
                <div class="form-group">
                    <label class="form-label">Entity *</label>
                    <select class="form-input" id="fieldEntity">
                        <option value="">Select entity...</option>
                        ${entityOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">Field Type *</label>
                    <select class="form-input" id="fieldType">
                        <option value="">Select type...</option>
                        ${typeOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label class="form-label">Name *</label>
                    <input type="text" class="form-input" id="fieldName" placeholder="e.g., First Name">
                </div>
                <div class="form-group">
                    <label class="form-label">Code *</label>
                    <input type="text" class="form-input" id="fieldCode" placeholder="e.g., first_name (lowercase, underscores)">
                </div>
                <div class="form-group">
                    <label class="form-label">
                        <input type="checkbox" id="fieldRequired"> Required field
                    </label>
                </div>
            ` + "`" + `;
            document.getElementById('modalFooter').innerHTML = ` + "`" + `
                <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
                <button class="btn btn-primary" onclick="createField()">Create Field</button>
            ` + "`" + `;
            document.getElementById('modalOverlay').classList.add('active');
        }

        async function createField() {
            const entity_id = document.getElementById('fieldEntity').value;
            const field_type_id = document.getElementById('fieldType').value;
            const name = document.getElementById('fieldName').value.trim();
            const code = document.getElementById('fieldCode').value.trim().toLowerCase();
            const is_required = document.getElementById('fieldRequired').checked;

            if (!entity_id || !field_type_id || !name || !code) {
                showToast('Entity, type, name and code are required', 'error');
                return;
            }

            try {
                await api('POST', '/admin/fields', {
                    entity_id, field_type_id, name, code, is_required
                });
                closeModal();
                showToast('Field created successfully', 'success');
                renderFields();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        async function deleteField(id) {
            if (!confirm('Are you sure you want to delete this field?')) return;
            try {
                await api('DELETE', '/admin/fields/' + id);
                showToast('Field deleted', 'success');
                renderFields();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // Users page
        async function renderUsers() {
            document.getElementById('headerActions').innerHTML = '<button class="btn btn-primary" onclick="openCreateUserModal()">+ New User</button>';
            document.getElementById('content').innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const users = await api('GET', '/admin/users?tenant_id=' + TENANT_ID);

                document.getElementById('content').innerHTML = ` + "`" + `
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Email</th>
                                    <th>Status</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody id="usersTable"></tbody>
                        </table>
                    </div>
                ` + "`" + `;

                const tbody = document.getElementById('usersTable');
                users.forEach(u => {
                    tbody.innerHTML += ` + "`" + `
                        <tr>
                            <td>${u.first_name} ${u.last_name}</td>
                            <td>${u.email}</td>
                            <td><span class="badge ${u.is_active ? 'badge-green' : 'badge-yellow'}">${u.is_active ? 'Active' : 'Inactive'}</span></td>
                            <td>
                                <button class="btn btn-secondary btn-sm">Edit</button>
                            </td>
                        </tr>
                    ` + "`" + `;
                });
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        function openCreateUserModal() {
            document.getElementById('modalTitle').textContent = 'Create User';
            document.getElementById('modalBody').innerHTML = ` + "`" + `
                <div class="form-group">
                    <label class="form-label">Email *</label>
                    <input type="email" class="form-input" id="userEmail" placeholder="user@example.com">
                </div>
                <div class="form-group">
                    <label class="form-label">Password *</label>
                    <input type="password" class="form-input" id="userPassword" placeholder="Min 8 characters">
                </div>
                <div class="form-group">
                    <label class="form-label">First Name</label>
                    <input type="text" class="form-input" id="userFirstName" placeholder="First name">
                </div>
                <div class="form-group">
                    <label class="form-label">Last Name</label>
                    <input type="text" class="form-input" id="userLastName" placeholder="Last name">
                </div>
            ` + "`" + `;
            document.getElementById('modalFooter').innerHTML = ` + "`" + `
                <button class="btn btn-secondary" onclick="closeModal()">Cancel</button>
                <button class="btn btn-primary" onclick="createUser()">Create User</button>
            ` + "`" + `;
            document.getElementById('modalOverlay').classList.add('active');
        }

        async function createUser() {
            const email = document.getElementById('userEmail').value.trim();
            const password = document.getElementById('userPassword').value;
            const first_name = document.getElementById('userFirstName').value.trim();
            const last_name = document.getElementById('userLastName').value.trim();

            if (!email || !password) {
                showToast('Email and password are required', 'error');
                return;
            }

            try {
                await api('POST', '/admin/users', {
                    tenant_id: TENANT_ID,
                    email, password, first_name, last_name
                });
                closeModal();
                showToast('User created successfully', 'success');
                renderUsers();
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // Tenants page
        async function renderTenants() {
            document.getElementById('headerActions').innerHTML = '';
            document.getElementById('content').innerHTML = '<div class="loading"><div class="spinner"></div></div>';

            try {
                const tenants = await api('GET', '/admin/tenants');

                document.getElementById('content').innerHTML = ` + "`" + `
                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Code</th>
                                    <th>Status</th>
                                </tr>
                            </thead>
                            <tbody id="tenantsTable"></tbody>
                        </table>
                    </div>
                ` + "`" + `;

                const tbody = document.getElementById('tenantsTable');
                tenants.forEach(t => {
                    tbody.innerHTML += ` + "`" + `
                        <tr>
                            <td>${t.name}</td>
                            <td><code>${t.code}</code></td>
                            <td><span class="badge ${t.is_active ? 'badge-green' : 'badge-yellow'}">${t.is_active ? 'Active' : 'Inactive'}</span></td>
                        </tr>
                    ` + "`" + `;
                });
            } catch (err) {
                showToast(err.message, 'error');
            }
        }

        // Modal functions
        function closeModal(event) {
            if (event && event.target !== event.currentTarget) return;
            document.getElementById('modalOverlay').classList.remove('active');
        }

        // Toast notifications
        function showToast(message, type = 'success') {
            const container = document.getElementById('toastContainer');
            const toast = document.createElement('div');
            toast.className = 'toast ' + type;
            toast.innerHTML = message;
            container.appendChild(toast);
            setTimeout(() => toast.remove(), 3000);
        }

        // Enter key support for login
        document.getElementById('loginPassword').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') doLogin();
        });
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}
