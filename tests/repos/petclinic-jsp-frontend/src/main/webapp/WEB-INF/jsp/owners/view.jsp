<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="${owner.fullName}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item active">${owner.fullName}</li>
    </ol>
</nav>

<div class="row">
    <div class="col-md-6">
        <div class="card owner-card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h4 class="mb-0"><i class="bi bi-person-circle me-2"></i>${owner.fullName}</h4>
                <div>
                    <a href="${pageContext.request.contextPath}/owners/${owner.id}/edit"
                       class="btn btn-sm btn-outline-primary">
                        <i class="bi bi-pencil me-1"></i>Edit
                    </a>
                    <button type="button" class="btn btn-sm btn-outline-danger"
                            onclick="confirmDelete(${owner.id}, '${owner.fullName}')">
                        <i class="bi bi-trash me-1"></i>Delete
                    </button>
                </div>
            </div>
            <div class="card-body">
                <table class="table table-borderless">
                    <tr>
                        <th width="30%"><i class="bi bi-geo-alt me-2"></i>Address</th>
                        <td>${owner.address}</td>
                    </tr>
                    <tr>
                        <th><i class="bi bi-building me-2"></i>City</th>
                        <td>${owner.city}</td>
                    </tr>
                    <tr>
                        <th><i class="bi bi-telephone me-2"></i>Telephone</th>
                        <td>${owner.telephone}</td>
                    </tr>
                    <c:if test="${not empty owner.email}">
                        <tr>
                            <th><i class="bi bi-envelope me-2"></i>Email</th>
                            <td>${owner.email}</td>
                        </tr>
                    </c:if>
                </table>
            </div>
        </div>
    </div>

    <div class="col-md-6">
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0"><i class="bi bi-heart me-2"></i>Pets</h5>
                <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/new"
                   class="btn btn-sm btn-success">
                    <i class="bi bi-plus-lg me-1"></i>Add Pet
                </a>
            </div>
            <div class="card-body">
                <c:choose>
                    <c:when test="${empty owner.pets}">
                        <div class="text-center text-muted py-3">
                            <i class="bi bi-heart display-6"></i>
                            <p class="mt-2">No pets registered yet.</p>
                        </div>
                    </c:when>
                    <c:otherwise>
                        <div class="list-group list-group-flush">
                            <c:forEach var="pet" items="${owner.pets}">
                                <div class="list-group-item">
                                    <div class="d-flex justify-content-between align-items-center">
                                        <div>
                                            <h6 class="mb-1">${pet.name}</h6>
                                            <small class="text-muted">
                                                ${pet.typeName} | Born: ${pet.birthDate} | Age: ${pet.age} years
                                            </small>
                                        </div>
                                        <div>
                                            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits/new"
                                               class="btn btn-sm btn-outline-primary" title="Add Visit">
                                                <i class="bi bi-calendar-plus"></i>
                                            </a>
                                            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits"
                                               class="btn btn-sm btn-outline-info" title="View Visits">
                                                <i class="bi bi-calendar-check"></i>
                                            </a>
                                            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/edit"
                                               class="btn btn-sm btn-outline-secondary" title="Edit Pet">
                                                <i class="bi bi-pencil"></i>
                                            </a>
                                        </div>
                                    </div>
                                </div>
                            </c:forEach>
                        </div>
                    </c:otherwise>
                </c:choose>
            </div>
        </div>
    </div>
</div>

<!-- Delete Confirmation Modal -->
<div class="modal fade" id="deleteModal" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header delete-modal">
                <h5 class="modal-title"><i class="bi bi-exclamation-triangle me-2"></i>Confirm Delete</h5>
                <button type="button" class="btn-close btn-close-white" data-bs-dismiss="modal"></button>
            </div>
            <div class="modal-body">
                <p>Are you sure you want to delete owner <strong id="deleteOwnerName"></strong>?</p>
                <p class="text-danger"><small>This action cannot be undone.</small></p>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                <button type="button" class="btn btn-danger" id="confirmDeleteBtn">Delete</button>
            </div>
        </div>
    </div>
</div>

<script>
    let ownerIdToDelete = null;

    function confirmDelete(id, name) {
        ownerIdToDelete = id;
        document.getElementById('deleteOwnerName').textContent = name;
        new bootstrap.Modal(document.getElementById('deleteModal')).show();
    }

    document.getElementById('confirmDeleteBtn').addEventListener('click', function() {
        if (ownerIdToDelete) {
            deleteOwner(ownerIdToDelete);
        }
    });

    function deleteOwner(id) {
        fetch('${pageContext.request.contextPath}/owners/' + id, {
            method: 'DELETE'
        }).then(response => {
            if (response.ok) {
                window.location.href = '${pageContext.request.contextPath}/owners';
            } else {
                alert('Failed to delete owner');
            }
        }).catch(error => {
            alert('Error: ' + error);
        });
    }
</script>

<jsp:include page="../includes/footer.jsp"/>
