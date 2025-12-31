<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="../includes/header.jsp">
    <jsp:param name="title" value="Veterinarians"/>
</jsp:include>

<div class="d-flex justify-content-between align-items-center mb-4">
    <h2><i class="bi bi-person-badge me-2"></i>Veterinarians</h2>
</div>

<c:choose>
    <c:when test="${empty vets}">
        <div class="empty-state">
            <i class="bi bi-person-badge"></i>
            <h4>No Veterinarians Found</h4>
            <p>There are no veterinarians in the system.</p>
        </div>
    </c:when>
    <c:otherwise>
        <div class="row g-4">
            <c:forEach var="vet" items="${vets}">
                <div class="col-md-4">
                    <div class="card h-100">
                        <div class="card-body">
                            <h5 class="card-title">
                                <i class="bi bi-person-circle text-primary me-2"></i>
                                <a href="${pageContext.request.contextPath}/vets/${vet.id}">
                                    ${vet.fullName}
                                </a>
                            </h5>
                            <div class="mt-3">
                                <strong>Specialties:</strong>
                                <div class="mt-2">
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
                                </div>
                            </div>
                        </div>
                        <div class="card-footer">
                            <a href="${pageContext.request.contextPath}/vets/${vet.id}"
                               class="btn btn-sm btn-outline-primary">
                                <i class="bi bi-eye me-1"></i>View Details
                            </a>
                        </div>
                    </div>
                </div>
            </c:forEach>
        </div>
    </c:otherwise>
</c:choose>

<jsp:include page="../includes/footer.jsp"/>
