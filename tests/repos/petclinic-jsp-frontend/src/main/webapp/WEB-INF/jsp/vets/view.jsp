<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="${vet.fullName}"/>
</jsp:include>

<nav aria-label="breadcrumb" class="mb-4">
    <ol class="breadcrumb">
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/home">Home</a></li>
        <li class="breadcrumb-item"><a href="${pageContext.request.contextPath}/vets">Veterinarians</a></li>
        <li class="breadcrumb-item active">${vet.fullName}</li>
    </ol>
</nav>

<div class="row justify-content-center">
    <div class="col-md-6">
        <div class="card">
            <div class="card-header">
                <h4 class="mb-0">
                    <i class="bi bi-person-badge text-primary me-2"></i>${vet.fullName}
                </h4>
            </div>
            <div class="card-body">
                <table class="table table-borderless">
                    <tr>
                        <th width="40%">First Name</th>
                        <td>${vet.firstName}</td>
                    </tr>
                    <tr>
                        <th>Last Name</th>
                        <td>${vet.lastName}</td>
                    </tr>
                    <tr>
                        <th>Specialties</th>
                        <td>
                            <c:choose>
                                <c:when test="${empty vet.specialties}">
                                    <span class="badge bg-secondary">None</span>
                                </c:when>
                                <c:otherwise>
                                    <c:forEach var="specialty" items="${vet.specialties}">
                                        <span class="badge bg-success specialty-badge">${specialty}</span>
                                    </c:forEach>
                                </c:otherwise>
                            </c:choose>
                        </td>
                    </tr>
                </table>
            </div>
            <div class="card-footer">
                <a href="${pageContext.request.contextPath}/vets" class="btn btn-outline-secondary">
                    <i class="bi bi-arrow-left me-1"></i>Back to Vets
                </a>
            </div>
        </div>
    </div>
</div>

<jsp:include page="../includes/footer.jsp"/>
