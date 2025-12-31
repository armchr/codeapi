<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="Pets - ${owner.fullName}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        <li class="breadcrumb-item active">Pets</li>
    </ol>
</nav>

<div class="d-flex justify-content-between align-items-center mb-4">
    <h2><i class="bi bi-heart me-2"></i>Pets for ${owner.fullName}</h2>
    <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/new" class="btn btn-success">
        <i class="bi bi-plus-lg me-1"></i>Add Pet
    </a>
</div>

<c:choose>
    <c:when test="${empty owner.pets}">
        <div class="empty-state">
            <i class="bi bi-heart"></i>
            <h4>No Pets Found</h4>
            <p>This owner doesn't have any pets registered yet.</p>
            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/new" class="btn btn-success">
                <i class="bi bi-plus-lg me-1"></i>Add First Pet
            </a>
        </div>
    </c:when>
    <c:otherwise>
        <div class="row g-4">
            <c:forEach var="pet" items="${owner.pets}">
                <div class="col-md-4">
                    <div class="card h-100">
                        <div class="card-body">
                            <h5 class="card-title">
                                <i class="bi bi-heart-fill text-danger me-2"></i>${pet.name}
                            </h5>
                            <p class="card-text">
                                <span class="badge bg-primary">${pet.typeName}</span>
                            </p>
                            <ul class="list-unstyled">
                                <li><strong>Birth Date:</strong> ${pet.birthDate}</li>
                                <li><strong>Age:</strong> ${pet.age} years</li>
                            </ul>
                        </div>
                        <div class="card-footer">
                            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits"
                               class="btn btn-sm btn-outline-info">
                                <i class="bi bi-calendar-check me-1"></i>Visits
                            </a>
                            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/edit"
                               class="btn btn-sm btn-outline-secondary">
                                <i class="bi bi-pencil me-1"></i>Edit
                            </a>
                        </div>
                    </div>
                </div>
            </c:forEach>
        </div>
    </c:otherwise>
</c:choose>

<jsp:include page="../includes/footer.jsp"/>
