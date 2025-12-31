<%@ page contentType="text/html;charset=UTF-8" isErrorPage="true" %>
<%@ taglib prefix="c" uri="jakarta.tags.core" %>
<jsp:include page="includes/header.jsp">
    <jsp:param name="title" value="Not Found"/>
</jsp:include>

<div class="row justify-content-center">
    <div class="col-md-6 text-center">
        <div class="display-1 text-muted mb-4">404</div>
        <h2>Page Not Found</h2>
        <p class="text-muted">The page you are looking for doesn't exist or has been moved.</p>
        <a href="${pageContext.request.contextPath}/home" class="btn btn-primary">
            <i class="bi bi-house me-1"></i>Go Home
        </a>
    </div>
</div>

<jsp:include page="includes/footer.jsp"/>
