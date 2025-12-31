<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="${action == 'create' ? 'New Owner' : 'Edit Owner'}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <c:if test="${action == 'edit'}">
            <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        </c:if>
        <li class="breadcrumb-item active">${action == 'create' ? 'New' : 'Edit'}</li>
    </ol>
</nav>

<div class="row justify-content-center">
    <div class="col-md-8">
        <div class="card">
            <div class="card-header">
                <h4 class="mb-0">
                    <i class="bi bi-${action == 'create' ? 'plus-circle' : 'pencil'} me-2"></i>
                    ${action == 'create' ? 'Add New Owner' : 'Edit Owner'}
                </h4>
            </div>
            <div class="card-body">
                <form action="${pageContext.request.contextPath}/owners${action == 'edit' ? '/'.concat(owner.id) : ''}"
                      method="post" class="needs-validation" novalidate>

                    <div class="row mb-3">
                        <div class="col-md-6">
                            <label for="firstName" class="form-label">First Name *</label>
                            <input type="text" class="form-control" id="firstName" name="firstName"
                                   value="${owner.firstName}" required minlength="1" maxlength="50">
                            <div class="invalid-feedback">First name is required.</div>
                        </div>
                        <div class="col-md-6">
                            <label for="lastName" class="form-label">Last Name *</label>
                            <input type="text" class="form-control" id="lastName" name="lastName"
                                   value="${owner.lastName}" required minlength="1" maxlength="50">
                            <div class="invalid-feedback">Last name is required.</div>
                        </div>
                    </div>

                    <div class="mb-3">
                        <label for="address" class="form-label">Address *</label>
                        <input type="text" class="form-control" id="address" name="address"
                               value="${owner.address}" required maxlength="255">
                        <div class="invalid-feedback">Address is required.</div>
                    </div>

                    <div class="row mb-3">
                        <div class="col-md-6">
                            <label for="city" class="form-label">City *</label>
                            <input type="text" class="form-control" id="city" name="city"
                                   value="${owner.city}" required maxlength="80">
                            <div class="invalid-feedback">City is required.</div>
                        </div>
                        <div class="col-md-6">
                            <label for="telephone" class="form-label">Telephone *</label>
                            <input type="tel" class="form-control" id="telephone" name="telephone"
                                   value="${owner.telephone}" required pattern="[0-9]{10}" maxlength="20">
                            <div class="invalid-feedback">Please enter a valid 10-digit telephone number.</div>
                        </div>
                    </div>

                    <div class="mb-3">
                        <label for="email" class="form-label">Email</label>
                        <input type="email" class="form-control" id="email" name="email"
                               value="${owner.email}" maxlength="100">
                        <div class="invalid-feedback">Please enter a valid email address.</div>
                    </div>

                    <div class="d-flex justify-content-between">
                        <a href="${pageContext.request.contextPath}/owners${action == 'edit' ? '/'.concat(owner.id) : ''}"
                           class="btn btn-secondary">
                            <i class="bi bi-x-lg me-1"></i>Cancel
                        </a>
                        <button type="submit" class="btn btn-primary">
                            <i class="bi bi-check-lg me-1"></i>${action == 'create' ? 'Create Owner' : 'Save Changes'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<script>
    // Form validation
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
