class OpenAPIValidator {
    constructor() {
        this.serverUrl = 'http://localhost:8080';
        this.validationMode = 'validate';
        this.currentSpec = null;
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadConfig();
    }

    bindEvents() {
        // Configuration
        document.getElementById('server-url').addEventListener('input', (e) => {
            this.serverUrl = e.target.value;
            this.saveConfig();
        });

        document.getElementById('validation-mode').addEventListener('change', (e) => {
            this.validationMode = e.target.value;
            this.updateValidateButton();
            this.saveConfig();
        });

        document.getElementById('test-connection').addEventListener('click', () => {
            this.testConnection();
        });

        // File upload
        const uploadArea = document.getElementById('upload-area');
        const fileInput = document.getElementById('file-input');

        uploadArea.addEventListener('click', () => fileInput.click());
        uploadArea.addEventListener('dragover', this.handleDragOver.bind(this));
        uploadArea.addEventListener('dragleave', this.handleDragLeave.bind(this));
        uploadArea.addEventListener('drop', this.handleDrop.bind(this));

        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.handleFile(e.target.files[0]);
            }
        });

        document.getElementById('clear-file').addEventListener('click', () => {
            this.clearFile();
        });

        // Text input
        document.getElementById('spec-input').addEventListener('input', (e) => {
            this.currentSpec = e.target.value.trim();
            this.updateValidateButton();
        });

        document.getElementById('load-example').addEventListener('click', () => {
            this.loadExample();
        });

        document.getElementById('clear-text').addEventListener('click', () => {
            document.getElementById('spec-input').value = '';
            this.currentSpec = null;
            this.updateValidateButton();
        });

        // Validation
        document.getElementById('validate-btn').addEventListener('click', () => {
            this.validateSpec();
        });

        // Results
        document.getElementById('export-results').addEventListener('click', () => {
            this.exportResults();
        });
    }

    loadConfig() {
        const savedConfig = localStorage.getItem('openapi-validator-config');
        if (savedConfig) {
            const config = JSON.parse(savedConfig);
            document.getElementById('server-url').value = config.serverUrl || this.serverUrl;
            document.getElementById('validation-mode').value = config.validationMode || this.validationMode;
            this.serverUrl = config.serverUrl || this.serverUrl;
            this.validationMode = config.validationMode || this.validationMode;
        }
        this.updateValidateButton();
    }

    saveConfig() {
        const config = {
            serverUrl: this.serverUrl,
            validationMode: this.validationMode
        };
        localStorage.setItem('openapi-validator-config', JSON.stringify(config));
    }

    async testConnection() {
        const statusEl = document.getElementById('connection-status');
        statusEl.className = 'status-indicator testing';
        statusEl.textContent = 'Testing connection...';

        try {
            const response = await fetch(`${this.serverUrl}/health`, {
                method: 'GET',
                headers: {
                    'Accept': 'application/json'
                }
            });

            if (response.ok) {
                const data = await response.json();
                statusEl.className = 'status-indicator success';
                statusEl.textContent = `✓ Connected to ${data.service} server (detailed: ${data.detailed})`;
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            statusEl.className = 'status-indicator error';
            statusEl.textContent = `✗ Connection failed: ${error.message}`;
        }
    }

    handleDragOver(e) {
        e.preventDefault();
        document.getElementById('upload-area').classList.add('dragover');
    }

    handleDragLeave(e) {
        e.preventDefault();
        document.getElementById('upload-area').classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        document.getElementById('upload-area').classList.remove('dragover');
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            this.handleFile(files[0]);
        }
    }

    async handleFile(file) {
        if (!this.isValidFileType(file)) {
            alert('Please select a valid OpenAPI file (.json, .yaml, .yml)');
            return;
        }

        try {
            const content = await this.readFile(file);
            this.currentSpec = content;
            this.showFileInfo(file);
            this.clearTextInput();
            this.updateValidateButton();
        } catch (error) {
            alert(`Error reading file: ${error.message}`);
        }
    }

    isValidFileType(file) {
        const validTypes = ['.json', '.yaml', '.yml'];
        return validTypes.some(type => file.name.toLowerCase().endsWith(type));
    }

    readFile(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = (e) => resolve(e.target.result);
            reader.onerror = (e) => reject(new Error('Failed to read file'));
            reader.readAsText(file);
        });
    }

    showFileInfo(file) {
        document.getElementById('file-name').textContent = file.name;
        document.getElementById('file-size').textContent = this.formatFileSize(file.size);
        document.getElementById('file-info').style.display = 'flex';
        document.getElementById('upload-area').style.display = 'none';
    }

    clearFile() {
        document.getElementById('file-info').style.display = 'none';
        document.getElementById('upload-area').style.display = 'block';
        document.getElementById('file-input').value = '';
        this.currentSpec = null;
        this.updateValidateButton();
    }

    clearTextInput() {
        document.getElementById('spec-input').value = '';
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    loadExample() {
        const exampleSpec = `openapi: 3.0.0
info:
  title: Pet Store API
  version: 1.0.0
  description: A sample API for managing pets
  contact:
    name: API Support
    email: support@petstore.com
servers:
  - url: https://api.petstore.com/v1
    description: Production server
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      description: Retrieve a list of all pets in the store
      tags:
        - pets
      parameters:
        - name: limit
          in: query
          description: Number of pets to return
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 100
            default: 20
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
        '400':
          description: Bad request
        '500':
          description: Internal server error
    post:
      operationId: createPet
      summary: Create a new pet
      description: Add a new pet to the store
      tags:
        - pets
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Pet created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
        '400':
          description: Invalid input
        '500':
          description: Internal server error
  /pets/{petId}:
    get:
      operationId: getPet
      summary: Get a specific pet
      description: Retrieve details of a specific pet by ID
      tags:
        - pets
      parameters:
        - name: petId
          in: path
          required: true
          description: The ID of the pet to retrieve
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: Pet details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
        '404':
          description: Pet not found
        '500':
          description: Internal server error

components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
          description: Unique identifier for the pet
        name:
          type: string
          description: Name of the pet
          minLength: 1
          maxLength: 100
        tag:
          type: string
          description: Tag for categorizing the pet
          maxLength: 50
    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
          description: Name of the pet
          minLength: 1
          maxLength: 100
        tag:
          type: string
          description: Tag for categorizing the pet
          maxLength: 50`;

        // Clear any file selection first
        this.clearFile();
        
        // Set the textarea value and update currentSpec
        const textInput = document.getElementById('spec-input');
        textInput.value = exampleSpec;
        this.currentSpec = exampleSpec;
        
        // Use setTimeout to ensure the value is set before updating the button
        setTimeout(() => {
            this.updateValidateButton();
        }, 0);
    }

    updateValidateButton() {
        const btn = document.getElementById('validate-btn');
        const textInput = document.getElementById('spec-input');
        
        // Check both currentSpec and the text input value
        const currentSpecText = this.currentSpec || textInput.value.trim();
        const hasSpec = currentSpecText && currentSpecText.length > 0;
        
        btn.disabled = !hasSpec;
        
        const btnText = document.querySelector('.btn-text');
        if (this.validationMode === 'lint') {
            btnText.textContent = 'Lint OpenAPI Spec';
        } else {
            btnText.textContent = 'Validate OpenAPI Spec';
        }
        
        // Ensure currentSpec is in sync
        if (hasSpec && !this.currentSpec) {
            this.currentSpec = textInput.value.trim();
        }
    }

    async validateSpec() {
        // Ensure currentSpec is up to date
        const textInput = document.getElementById('spec-input');
        if (!this.currentSpec && textInput.value.trim()) {
            this.currentSpec = textInput.value.trim();
        }
        
        if (!this.currentSpec) {
            alert('Please provide an OpenAPI specification to validate');
            return;
        }

        const spec = this.currentSpec;

        this.showLoading(true);
        this.hideResults();

        try {
            const endpoint = this.validationMode === 'lint' ? '/lint' : '/validate';
            const response = await fetch(`${this.serverUrl}${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    openapi_spec: spec
                })
            });

            const result = await response.json();
            this.displayResults(result);
        } catch (error) {
            this.displayError(`Failed to validate specification: ${error.message}`);
        } finally {
            this.showLoading(false);
        }
    }

    showLoading(show) {
        const btn = document.getElementById('validate-btn');
        const btnText = document.querySelector('.btn-text');
        const btnSpinner = document.querySelector('.btn-spinner');
        
        btn.disabled = show;
        btnText.style.display = show ? 'none' : 'inline';
        btnSpinner.style.display = show ? 'inline' : 'none';
    }

    hideResults() {
        document.getElementById('results-section').style.display = 'none';
    }

    displayResults(result) {
        const resultsSection = document.getElementById('results-section');
        const resultsSummary = document.getElementById('results-summary');
        const resultsContent = document.getElementById('results-content');
        
        this.lastResults = result;

        // Show results section
        resultsSection.style.display = 'block';

        // Create summary
        resultsSummary.innerHTML = '';
        
        if (result.success) {
            resultsSummary.innerHTML = `
                <div class="summary-item summary-success">
                    ✓ Validation Passed
                </div>
            `;
        } else {
            if (result.error_count > 0) {
                resultsSummary.innerHTML += `
                    <div class="summary-item summary-error">
                        ${result.error_count} Error${result.error_count !== 1 ? 's' : ''}
                    </div>
                `;
            }
            if (result.warning_count > 0) {
                resultsSummary.innerHTML += `
                    <div class="summary-item summary-warning">
                        ${result.warning_count} Warning${result.warning_count !== 1 ? 's' : ''}
                    </div>
                `;
            }
        }

        // Create content
        if (result.success && (!result.issues || result.issues.length === 0)) {
            resultsContent.innerHTML = `
                <div class="success-message">
                    <div class="success-icon">✅</div>
                    <h3>Validation Successful!</h3>
                    <p>Your OpenAPI specification is valid and follows best practices.</p>
                </div>
            `;
        } else {
            resultsContent.innerHTML = `
                <div class="issues-list">
                    ${result.issues.map(issue => this.createIssueHTML(issue)).join('')}
                </div>
            `;
        }

        // Scroll to results
        resultsSection.scrollIntoView({ behavior: 'smooth' });
    }

    createIssueHTML(issue) {
        const contextParts = [];
        if (issue.operation) contextParts.push(`Operation: ${issue.operation}`);
        if (issue.path) contextParts.push(`Path: ${issue.path}`);
        if (issue.method) contextParts.push(`Method: ${issue.method.toUpperCase()}`);
        if (issue.parameter) contextParts.push(`Parameter: ${issue.parameter}`);
        if (issue.field) contextParts.push(`Field: ${issue.field}`);

        return `
            <div class="issue-item issue-${issue.type}">
                <div class="issue-header">
                    <div class="issue-type ${issue.type}">${issue.type}</div>
                    <div class="issue-message">${this.escapeHtml(issue.message)}</div>
                </div>
                ${issue.suggestion ? `<div class="issue-suggestion">💡 ${this.escapeHtml(issue.suggestion)}</div>` : ''}
                ${contextParts.length > 0 ? `
                    <div class="issue-context">
                        ${contextParts.map(part => `<span>${this.escapeHtml(part)}</span>`).join('')}
                    </div>
                ` : ''}
            </div>
        `;
    }

    displayError(message) {
        const resultsSection = document.getElementById('results-section');
        const resultsContent = document.getElementById('results-content');
        const resultsSummary = document.getElementById('results-summary');

        resultsSection.style.display = 'block';
        resultsSummary.innerHTML = `
            <div class="summary-item summary-error">
                ✗ Validation Failed
            </div>
        `;

        resultsContent.innerHTML = `
            <div class="issue-item issue-error">
                <div class="issue-header">
                    <div class="issue-type error">Error</div>
                    <div class="issue-message">${this.escapeHtml(message)}</div>
                </div>
                <div class="issue-suggestion">
                    💡 Check your server configuration and ensure the OpenAPI validation service is running.
                </div>
            </div>
        `;

        resultsSection.scrollIntoView({ behavior: 'smooth' });
    }

    exportResults() {
        if (!this.lastResults) {
            alert('No results to export');
            return;
        }

        const dataStr = JSON.stringify(this.lastResults, null, 2);
        const dataBlob = new Blob([dataStr], { type: 'application/json' });
        
        const link = document.createElement('a');
        link.href = URL.createObjectURL(dataBlob);
        link.download = `openapi-validation-results-${new Date().toISOString().slice(0, 19).replace(/:/g, '-')}.json`;
        link.click();
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the validator when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new OpenAPIValidator();
});