/**
 * PetClinic JSP Frontend - JavaScript Module
 * Handles AJAX operations and client-side interactions
 */

const PetClinic = {
    /**
     * Initialize the application
     */
    init: function() {
        this.setupDeleteConfirmations();
        this.setupFormValidation();
        this.setupAjaxSearch();
    },

    /**
     * Setup delete confirmation dialogs for AJAX deletes
     */
    setupDeleteConfirmations: function() {
        document.querySelectorAll('[data-delete-confirm]').forEach(function(element) {
            element.addEventListener('click', function(e) {
                e.preventDefault();
                const url = this.getAttribute('data-delete-url');
                const name = this.getAttribute('data-delete-name');
                const redirectUrl = this.getAttribute('data-delete-redirect');

                if (confirm('Are you sure you want to delete "' + name + '"?')) {
                    PetClinic.deleteResource(url, redirectUrl);
                }
            });
        });
    },

    /**
     * Delete a resource via AJAX
     */
    deleteResource: function(url, redirectUrl) {
        fetch(url, {
            method: 'DELETE',
            headers: {
                'Accept': 'application/json'
            }
        })
        .then(function(response) {
            if (response.ok) {
                if (redirectUrl) {
                    window.location.href = redirectUrl;
                } else {
                    window.location.reload();
                }
            } else {
                throw new Error('Delete failed: ' + response.status);
            }
        })
        .catch(function(error) {
            PetClinic.showAlert('Error: ' + error.message, 'danger');
        });
    },

    /**
     * Setup form validation
     */
    setupFormValidation: function() {
        var forms = document.querySelectorAll('.needs-validation');
        Array.prototype.slice.call(forms).forEach(function(form) {
            form.addEventListener('submit', function(event) {
                if (!form.checkValidity()) {
                    event.preventDefault();
                    event.stopPropagation();
                }
                form.classList.add('was-validated');
            }, false);
        });
    },

    /**
     * Setup AJAX search functionality
     */
    setupAjaxSearch: function() {
        var searchInput = document.getElementById('ajaxSearch');
        if (searchInput) {
            var debounceTimer;
            searchInput.addEventListener('input', function() {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(function() {
                    PetClinic.performSearch(searchInput.value);
                }, 300);
            });
        }
    },

    /**
     * Perform AJAX search
     */
    performSearch: function(query) {
        var resultsContainer = document.getElementById('searchResults');
        if (!resultsContainer) return;

        if (query.length < 2) {
            resultsContainer.innerHTML = '';
            return;
        }

        var searchUrl = resultsContainer.getAttribute('data-search-url');
        fetch(searchUrl + '?q=' + encodeURIComponent(query), {
            headers: {
                'Accept': 'application/json'
            }
        })
        .then(function(response) {
            return response.json();
        })
        .then(function(data) {
            PetClinic.displaySearchResults(data, resultsContainer);
        })
        .catch(function(error) {
            console.error('Search error:', error);
        });
    },

    /**
     * Display search results
     */
    displaySearchResults: function(results, container) {
        if (results.length === 0) {
            container.innerHTML = '<p class="text-muted">No results found.</p>';
            return;
        }

        var html = '<ul class="list-group">';
        results.forEach(function(item) {
            html += '<li class="list-group-item">';
            html += '<a href="' + item.url + '">' + PetClinic.escapeHtml(item.name) + '</a>';
            if (item.description) {
                html += '<br><small class="text-muted">' + PetClinic.escapeHtml(item.description) + '</small>';
            }
            html += '</li>';
        });
        html += '</ul>';
        container.innerHTML = html;
    },

    /**
     * Show an alert message
     */
    showAlert: function(message, type) {
        type = type || 'info';
        var alertDiv = document.createElement('div');
        alertDiv.className = 'alert alert-' + type + ' alert-dismissible fade show';
        alertDiv.setAttribute('role', 'alert');
        alertDiv.innerHTML = message +
            '<button type="button" class="btn-close" data-bs-dismiss="alert"></button>';

        var container = document.querySelector('main.container');
        if (container) {
            container.insertBefore(alertDiv, container.firstChild);
        }

        // Auto-dismiss after 5 seconds
        setTimeout(function() {
            if (alertDiv.parentNode) {
                alertDiv.remove();
            }
        }, 5000);
    },

    /**
     * Escape HTML to prevent XSS
     */
    escapeHtml: function(text) {
        var div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    /**
     * Format a date for display
     */
    formatDate: function(dateString) {
        if (!dateString) return '';
        var date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    },

    /**
     * Show loading spinner
     */
    showLoading: function(container) {
        container.innerHTML = '<div class="loading-spinner"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Loading...</span></div></div>';
    },

    /**
     * AJAX form submission
     */
    submitFormAjax: function(form, successCallback) {
        var formData = new FormData(form);
        var url = form.action;
        var method = form.method.toUpperCase();

        fetch(url, {
            method: method,
            body: formData
        })
        .then(function(response) {
            if (!response.ok) {
                throw new Error('Form submission failed');
            }
            return response.json();
        })
        .then(function(data) {
            if (successCallback) {
                successCallback(data);
            }
        })
        .catch(function(error) {
            PetClinic.showAlert('Error: ' + error.message, 'danger');
        });
    }
};

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', function() {
    PetClinic.init();
});
