<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="Owners"/>
</jsp:include>

<div class="d-flex justify-content-between align-items-center mb-4">
    <h2><i class="bi bi-people me-2"></i>Pet Owners</h2>
    <a href="${pageContext.request.contextPath}/owners/new" class="btn btn-primary">
        <i class="bi bi-plus-lg me-1"></i>Add Owner
    </a>
</div>

<!-- Search Form -->
<div class="card mb-4">
    <div class="card-body">
        <form action="${pageContext.request.contextPath}/owners" method="get" class="row g-3">
            <div class="col-md-9">
                <input type="text" class="form-control" name="lastName"
                       value="${searchTerm}" placeholder="Search by last name...">
            </div>
            <div class="col-md-3">
                <button type="submit" class="btn btn-outline-primary w-100">
                    <i class="bi bi-search me-1"></i>Search
                </button>
            </div>
        </form>
    </div>
</div>

<c:choose>
    <c:when test="${empty owners}">
        <div class="empty-state">
            <i class="bi bi-inbox"></i>
            <h4>No Owners Found</h4>
            <p>
                <c:choose>
                    <c:when test="${not empty searchTerm}">
                        No owners matching "${searchTerm}" were found.
                    </c:when>
                    <c:otherwise>
                        There are no owners in the system yet.
                    </c:otherwise>
                </c:choose>
            </p>
            <a href="${pageContext.request.contextPath}/owners/new" class="btn btn-primary">
                <i class="bi bi-plus-lg me-1"></i>Add First Owner
            </a>
        </div>
    </c:when>
    <c:otherwise>
        <div class="table-responsive">
            <table class="table table-hover">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Address</th>
                        <th>City</th>
                        <th>Telephone</th>
                        <th>Pets</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <c:forEach var="owner" items="${owners}">
                        <tr>
                            <td>
                                <a href="${pageContext.request.contextPath}/owners/${owner.id}">
                                    ${owner.fullName}
                                </a>
                            </td>
                            <td>${owner.address}</td>
                            <td>${owner.city}</td>
                            <td>${owner.telephone}</td>
                            <td>
                                <c:forEach var="pet" items="${owner.pets}" varStatus="status">
                                    <span class="badge bg-info pet-badge">${pet.name}</span>
                                </c:forEach>
                                <c:if test="${empty owner.pets}">
                                    <span class="text-muted">No pets</span>
                                </c:if>
                            </td>
                            <td>
                                <a href="${pageContext.request.contextPath}/owners/${owner.id}"
                                   class="btn btn-sm btn-outline-primary btn-action" title="View">
                                    <i class="bi bi-eye"></i>
                                </a>
                                <a href="${pageContext.request.contextPath}/owners/${owner.id}/edit"
                                   class="btn btn-sm btn-outline-secondary btn-action" title="Edit">
                                    <i class="bi bi-pencil"></i>
                                </a>
                            </td>
                        </tr>
                    </c:forEach>
                </tbody>
            </table>
        </div>
    </c:otherwise>
</c:choose>

<jsp:include page="../includes/footer.jsp"/>
