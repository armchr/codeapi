<%@ page contentType="text/html;charset=UTF-8" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="includes/header.jsp">
    <jsp:param name="title" value="Error"/>
</jsp:include>

<div class="row justify-content-center">
    <div class="col-md-6">
        <div class="card border-danger">
            <div class="card-header bg-danger text-white">
                <h4 class="mb-0"><i class="bi bi-exclamation-triangle me-2"></i>Error</h4>
            </div>
            <div class="card-body">
                <p class="card-text">${error}</p>
                <a href="${pageContext.request.contextPath}/home" class="btn btn-primary">
                    <i class="bi bi-house me-1"></i>Go Home
                </a>
            </div>
        </div>
    </div>
</div>

<jsp:include page="includes/footer.jsp"/>
