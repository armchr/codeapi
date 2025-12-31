<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="New Visit"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits">Visits for ${pet.name}</a></li>
        <li class="breadcrumb-item active">New Visit</li>
    </ol>
</nav>

<div class="row justify-content-center">
    <div class="col-md-6">
        <div class="card">
            <div class="card-header">
                <h4 class="mb-0">
                    <i class="bi bi-calendar-plus me-2"></i>New Visit for ${pet.name}
                </h4>
            </div>
            <div class="card-body">
                <div class="mb-3 p-3 bg-light rounded">
                    <p class="mb-1"><strong>Pet:</strong> ${pet.name}</p>
                    <p class="mb-1"><strong>Type:</strong> <span class="badge bg-primary">${pet.typeName}</span></p>
                    <p class="mb-0"><strong>Owner:</strong> ${owner.fullName}</p>
                </div>

                <form action="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits"
                      method="post" class="needs-validation" novalidate>

                    <div class="mb-3">
                        <label for="date" class="form-label">Visit Date *</label>
                        <input type="date" class="form-control" id="date" name="date"
                               value="${visit.date}" required>
                        <div class="invalid-feedback">Visit date is required.</div>
                    </div>

                    <div class="mb-3">
                        <label for="description" class="form-label">Description *</label>
                        <textarea class="form-control" id="description" name="description"
                                  rows="4" required maxlength="500"
                                  placeholder="Describe the reason for visit...">${visit.description}</textarea>
                        <div class="invalid-feedback">Description is required.</div>
                    </div>

                    <div class="d-flex justify-content-between">
                        <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits"
                           class="btn btn-secondary">
                            <i class="bi bi-x-lg me-1"></i>Cancel
                        </a>
                        <button type="submit" class="btn btn-success">
                            <i class="bi bi-check-lg me-1"></i>Add Visit
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<script>
    // Set default date to today
    document.addEventListener('DOMContentLoaded', function() {
        var dateInput = document.getElementById('date');
        if (!dateInput.value) {
            dateInput.valueAsDate = new Date();
        }
    });

    (function () {
        'use strict'
        var forms = document.querySelectorAll('.needs-validation')
        Array.prototype.slice.call(forms).forEach(function (form) {
            form.addEventListener('submit', function (event) {
                if (!form.checkValidity()) {
                    event.preventDefault()
                    event.stopPropagation()
                }
                form.classList.add('was-validated')
            }, false)
        })
    })()
</script>

<jsp:include page="../includes/footer.jsp"/>
