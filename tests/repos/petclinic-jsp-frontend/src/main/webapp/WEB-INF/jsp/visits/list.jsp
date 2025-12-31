<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="Visits - ${pet.name}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners">Owners</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/owners/${owner.id}">${owner.fullName}</a></li>
        <li class="breadcrumb-item active">Visits for ${pet.name}</li>
    </ol>
</nav>

<div class="d-flex justify-content-between align-items-center mb-4">
    <div>
        <h2><i class="bi bi-calendar-check me-2"></i>Visits for ${pet.name}</h2>
        <p class="text-muted mb-0">
            <span class="badge bg-primary">${pet.typeName}</span>
            Owner: ${owner.fullName}
        </p>
    </div>
    <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits/new"
       class="btn btn-success">
        <i class="bi bi-plus-lg me-1"></i>Add Visit
    </a>
</div>

<c:choose>
    <c:when test="${empty visits}">
        <div class="empty-state">
            <i class="bi bi-calendar-x"></i>
            <h4>No Visits Found</h4>
            <p>No visits have been recorded for ${pet.name} yet.</p>
            <a href="${pageContext.request.contextPath}/owners/${owner.id}/pets/${pet.id}/visits/new"
               class="btn btn-success">
                <i class="bi bi-plus-lg me-1"></i>Schedule First Visit
            </a>
        </div>
    </c:when>
    <c:otherwise>
        <div class="card">
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-hover">
                        <thead>
                            <tr>
                                <th>Date</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <c:forEach var="visit" items="${visits}">
                                <tr>
                                    <td>
                                        <i class="bi bi-calendar3 me-1"></i>
                                        ${visit.date}
                                    </td>
                                    <td>${visit.description}</td>
                                </tr>
                            </c:forEach>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </c:otherwise>
</c:choose>

<div class="mt-3">
    <a href="${pageContext.request.contextPath}/owners/${owner.id}" class="btn btn-outline-secondary">
        <i class="bi bi-arrow-left me-1"></i>Back to Owner
    </a>
</div>

<jsp:include page="../includes/footer.jsp"/>
