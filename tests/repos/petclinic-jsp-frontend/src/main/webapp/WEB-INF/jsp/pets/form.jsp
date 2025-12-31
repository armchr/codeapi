<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="${action == 'create' ? 'New Pet' : 'Edit Pet'}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        <li class="breadcrumb-item active">${action == 'create' ? 'New Pet' : 'Edit Pet'}</li>
    </ol>
</nav>

<div class="row justify-content-center">
    <div class="col-md-6">
        <div class="card">
            <div class="card-header">
                <h4 class="mb-0">
                    <i class="bi bi-${action == 'create' ? 'plus-circle' : 'pencil'} me-2"></i>
                    ${action == 'create' ? 'Add New Pet' : 'Edit Pet'}
                </h4>
            </div>
            <div class="card-body">
                <p class="text-muted">Owner: <strong>${owner.fullName}</strong></p>

                <form action="${pageContext.request.contextPath}/owners/${owner.id}/pets${action == 'edit' ? '/'.concat(pet.id) : ''}"
                      method="post" class="needs-validation" novalidate>

                    <div class="mb-3">
                        <label for="name" class="form-label">Pet Name *</label>
                        <input type="text" class="form-control" id="name" name="name"
                               value="${pet.name}" required maxlength="50">
                        <div class="invalid-feedback">Pet name is required.</div>
                    </div>

                    <div class="mb-3">
                        <label for="birthDate" class="form-label">Birth Date *</label>
                        <input type="date" class="form-control" id="birthDate" name="birthDate"
                               value="${pet.birthDate}" required>
                        <div class="invalid-feedback">Birth date is required.</div>
                    </div>

                    <div class="mb-3">
                        <label for="typeName" class="form-label">Pet Type *</label>
                        <select class="form-select" id="typeName" name="typeName" required>
                            <option value="">Select a type...</option>
                            <c:forEach var="petType" items="${petTypes}">
                                <option value="${petType.name}"
                                    ${pet.typeName == petType.name ? 'selected' : ''}>
                                    ${petType.name}
                                </option>
                            </c:forEach>
                        </select>
                        <div class="invalid-feedback">Please select a pet type.</div>
                    </div>

                    <div class="d-flex justify-content-between">
                        <a href="${pageContext.request.contextPath}/owners/${owner.id}"
                           class="btn btn-secondary">
                            <i class="bi bi-x-lg me-1"></i>Cancel
                        </a>
                        <button type="submit" class="btn btn-success">
                            <i class="bi bi-check-lg me-1"></i>${action == 'create' ? 'Add Pet' : 'Save Changes'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<script>
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
